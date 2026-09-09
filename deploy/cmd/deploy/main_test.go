package main

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRollout(t *testing.T) {
	old := repository + "sha256:" + strings.Repeat("a", 64)
	digest := "sha256:" + strings.Repeat("b", 64)
	for _, tc := range []struct {
		name                                 string
		failDeploy, failVerify, failRollback bool
		calls                                int
		want                                 string
	}{
		{name: "success", calls: 1}, {name: "deploy failure", failDeploy: true, calls: 2, want: "previous image restored"}, {name: "verification failure", failVerify: true, calls: 2, want: "previous image restored"}, {name: "rollback failure", failDeploy: true, failRollback: true, calls: 2, want: "ROLLBACK FAILED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var images []string
			r := func(_ time.Duration, _ string, args ...string) (string, error) {
				if args[0] == "git:report" {
					return old, nil
				}
				images = append(images, args[2])
				if len(images) == 1 && tc.failDeploy || len(images) == 2 && tc.failRollback {
					return "", errors.New("failed")
				}
				return "", nil
			}
			v := func(image string) error {
				if image != old && tc.failVerify {
					return errors.New("unhealthy")
				}
				return nil
			}
			err := rollout(r, v, digest)
			if len(images) != tc.calls {
				t.Fatalf("calls: %v", images)
			}
			if len(images) == 2 && images[1] != old {
				t.Fatal("wrong rollback image")
			}
			if tc.want == "" && err != nil {
				t.Fatal(err)
			}
			if tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("unexpected error %v", err)
			}
		})
	}
}
func TestRefusesUnsafeInput(t *testing.T) {
	for _, d := range []string{"latest", "sha256:abc", "sha256:" + strings.Repeat("a", 64) + "; reboot"} {
		called := false
		err := rollout(func(time.Duration, string, ...string) (string, error) { called = true; return "", nil }, func(string) error { return nil }, d)
		if err == nil || called {
			t.Fatal("unsafe input accepted")
		}
	}
}
func TestNoRollbackTargetOrUnhealthyBaseline(t *testing.T) {
	for _, previous := range []string{"codeforafrica/other:latest", repository + "sha256:" + strings.Repeat("a", 64)} {
		calls := 0
		err := rollout(func(time.Duration, string, ...string) (string, error) { calls++; return previous, nil }, func(string) error { return errors.New("bad baseline") }, "sha256:"+strings.Repeat("b", 64))
		if err == nil || calls != 1 {
			t.Fatal("changed deployment without valid baseline")
		}
	}
}
