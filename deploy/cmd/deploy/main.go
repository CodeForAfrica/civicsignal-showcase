package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const app = "civicsignal-web"
const repository = "codeforafrica/civicsignal-web@"

var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

type runner func(time.Duration, string, ...string) (string, error)

func run(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	// Kill the entire subprocess group on timeout, before attempting rollback.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 5 * time.Second
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s: %w: %s", name, err, out)
	}
	return string(out), nil
}
func deployImage(r runner, image string) error {
	out, err := r(12*time.Minute, "/usr/bin/dokku", "git:from-image", app, image)
	fmt.Print(out)
	return err
}
func rollout(r runner, verify func(string) error, digest string) error {
	if !digestPattern.MatchString(digest) {
		return errors.New("invalid image digest")
	}
	previous, err := r(time.Minute, "/usr/bin/dokku", "git:report", app, "--git-source-image")
	if err != nil {
		return err
	}
	previous = strings.TrimSpace(previous)
	if !strings.HasPrefix(previous, repository) || !digestPattern.MatchString(strings.TrimPrefix(previous, repository)) {
		return errors.New("previous deployment is not a pinned CivicSignal image; refusing without rollback target")
	}
	// Refuse to change an already-unhealthy application.
	if err = verify(previous); err != nil {
		return fmt.Errorf("baseline check failed: %w", err)
	}
	fmt.Println("Rollback target:", previous)
	image := repository + digest
	if err = deployImage(r, image); err == nil {
		err = verify(image)
	}
	if err == nil {
		fmt.Println("Deployment verified:", image)
		return nil
	}
	fmt.Println("Deployment failed; restoring", previous)
	if restoreErr := deployImage(r, previous); restoreErr != nil {
		return fmt.Errorf("DEPLOY FAILED: %v; ROLLBACK FAILED: %w", err, restoreErr)
	}
	if restoreErr := verify(previous); restoreErr != nil {
		return fmt.Errorf("DEPLOY FAILED: %v; ROLLBACK VERIFICATION FAILED: %w", err, restoreErr)
	}
	return fmt.Errorf("deployment failed; previous image restored: %w", err)
}
func verify(image string) error {
	// Compare running container image with the image Dokku built, not just an old HTTP response.
	built, err := run(time.Minute, "/usr/bin/docker", "image", "inspect", "dokku/"+app+":latest", "--format", "{{.Id}}")
	if err != nil {
		return err
	}
	var last error
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		actual, e := run(10*time.Second, "/usr/bin/docker", "inspect", app+".web.1", "--format", "{{.Image}} {{.State.Health.Status}}")
		if e == nil && strings.TrimSpace(actual) == strings.TrimSpace(built)+" healthy" {
			revision, e := run(10*time.Second, "/usr/bin/dokku", "git:report", app, "--git-source-image")
			if e == nil && strings.TrimSpace(revision) == image {
				if e = checkHTTP(); e == nil {
					return nil
				} else {
					last = e
				}
			} else {
				last = errors.New("deployed image reference does not match")
			}
		} else {
			last = errors.New("container not running the expected healthy image")
		}
		time.Sleep(5 * time.Second)
	}
	return last
}
func checkHTTP() error {
	client := http.Client{Timeout: 15 * time.Second}
	for _, p := range []string{"healthz", "index.html", "login.html", "showcase_data.json"} {
		res, err := client.Get("https://civicsignal.africa/" + p)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, 20<<20))
		res.Body.Close()
		if err != nil || res.StatusCode != 200 {
			return fmt.Errorf("HTTP check failed for %s: %d", p, res.StatusCode)
		}
		if p == "healthz" && strings.TrimSpace(string(body)) != "ok" {
			return errors.New("invalid health response")
		}
		if p == "showcase_data.json" {
			var d struct {
				GeneratedAt string
				Publishers  []json.RawMessage
			}
			if json.Unmarshal(body, &d) != nil || d.GeneratedAt == "" || len(d.Publishers) == 0 {
				return errors.New("invalid published dataset")
			}
		}
		if p == "index.html" && !strings.Contains(string(body), "https://tools.civicsignal.africa/#/login") {
			return errors.New("portal login link missing")
		}
	}
	return nil
}
func dispatch(digest string) error {
	if !digestPattern.MatchString(digest) {
		return errors.New("invalid digest")
	}
	params, _ := json.Marshal(map[string][]string{"ImageDigest": {digest}})
	out, err := run(time.Minute, "aws", "ssm", "send-command", "--region", "eu-west-1", "--instance-ids", "i-046fe9e345ae1424e", "--document-name", "civicsignal-web-deploy", "--document-version", "1", "--parameters", string(params), "--timeout-seconds", "120", "--output", "json")
	if err != nil {
		return err
	}
	var sent struct{ Command struct{ CommandId string } }
	if json.Unmarshal([]byte(out), &sent) != nil || sent.Command.CommandId == "" {
		return errors.New("missing SSM command ID")
	}
	id := sent.Command.CommandId
	fmt.Println("SSM command:", id)
	// Polling is read-only. Do not cancel SSM on workflow cancellation: let rollback finish.
	deadline := time.Now().Add(40 * time.Minute)
	for time.Now().Before(deadline) {
		out, e := run(time.Minute, "aws", "ssm", "get-command-invocation", "--region", "eu-west-1", "--command-id", id, "--instance-id", "i-046fe9e345ae1424e", "--output", "json")
		if e != nil {
			if !strings.Contains(out, "InvocationDoesNotExist") {
				return e
			}
			time.Sleep(10 * time.Second)
			continue
		}
		var result struct{ Status, StandardOutputContent, StandardErrorContent string }
		if e = json.Unmarshal([]byte(out), &result); e != nil {
			return e
		}
		switch result.Status {
		case "Pending", "InProgress", "Delayed":
			time.Sleep(10 * time.Second)
		case "Success":
			fmt.Print(result.StandardOutputContent)
			return nil
		default:
			return fmt.Errorf("SSM %s: %s\n%s", result.Status, result.StandardOutputContent, result.StandardErrorContent)
		}
	}
	return fmt.Errorf("SSM wait expired; inspect command %s before starting another deployment", id)
}
func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: deploy server|dispatch sha256:DIGEST")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "server":
		if os.Geteuid() != 0 {
			err = errors.New("server helper requires root")
			break
		}
		lock, e := os.OpenFile("/run/lock/civicsignal-web-deploy.lock", os.O_CREATE|os.O_RDWR, 0600)
		if e != nil {
			err = e
			break
		}
		defer lock.Close()
		if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
			err = errors.New("another deployment is running")
			break
		}
		err = rollout(run, verify, os.Args[2])
	case "dispatch":
		err = dispatch(os.Args[2])
	default:
		err = errors.New("unknown mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
