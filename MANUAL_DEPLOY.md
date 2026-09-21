> A manual GitHub Action using AWS OIDC/SSM is available in code.
> Complete the [one-time setup](deploy/README.md) before using it.
> This document remains the operator fallback and domain-cutover reference.

# CivicSignal website: manual deployment and rollback

Prepared 9 September 2026 for Dokku 0.37.6 on `34.250.120.147`
(`i-046fe9e345ae1424e`, eu-west-1). The first production deployment completed on 9 September 2026.
See [RELEASE.md](RELEASE.md) for the current image and verification record.
Stages 2–4 describe first-time setup: do not recreate the app or repeat the
domain transfer for ordinary revisions. Run each required stage separately.

## Release status

- Deployed release: `codeforafrica/civicsignal-web:2026-09-09-02`, linux/arm64.
- Source: showcase commit `b3d49a9` (rendered upstream plus container and portal fixes).
- Candidate has passed local static-page/browser checks.
- The rendered repository includes `showcase_data.json`, dated 9 September 2026.
  Use this repository as the image build context; do not build from the source
  repository’s incomplete `page/` output.
- Live domains: `civicsignal.africa`, `www.civicsignal.africa` and
  `preview.civicsignal.africa`, served by `civicsignal-web`.
- Portal remains on `civicsignal-tools`. The new preview DNS record is the only
  DNS addition. Apex and www DNS targets were unchanged.

## Boundaries and rollback baseline

| App | Domains before launch | Treatment |
| --- | --- | --- |
| civicsignal-tools | tools.civicsignal.africa, civicsignal.africa | Keep running; move only the apex hostname |
| civicsignal-explorer | explorer.civicsignal.africa | No changes |
| civicsignal-sources | sources.civicsignal.africa | No changes |
| civicsignal-topics | topics.civicsignal.africa | No changes |

The old website uses `codeforafrica/civicsignal-web-tools:3.20.3`, digest
`sha256:a301d693b7f6f88011ee1e30f0419117db0ebcccb2f3c97ddcc38040311d2540`.
Its portal stays at `https://tools.civicsignal.africa/#/login`. Do not replace its
image, copy its environment into the new app, link its databases, remove its
storage, or run global rebuild/restart/prune commands.

Cloudflare apex and www records already resolve to this server without proxying.
The old certificate covers apex and tools, NOT www. It expires 18 November 2026.
Recheck these facts at execution time.

## 1. Finalize and publish the tested image — local machine

Rebuild and recreate the local preview as described in
CONTAINER.md. Run the browser suite and confirm exit 0. Successful HTML responses
alone do not establish that the data-dependent pages work; inspect their charts
and controls with the real dataset. If an authorized test account is available, verify a real portal login;
automated redirect checks do not authenticate. The initial release verified
unchanged portal HTML and container identity, not an authenticated session.

Save the source changes in version control before the final release build. Use a
new tag if this candidate changes; do not silently overwrite a published tag.

```sh
docker build --platform linux/arm64 -t codeforafrica/civicsignal-web:2026-09-09-02 .
docker login --username YOUR_DOCKER_HUB_USERNAME
docker push codeforafrica/civicsignal-web:2026-09-09-02
docker buildx imagetools inspect codeforafrica/civicsignal-web:2026-09-09-02
```

Enter the password/token at the prompt, never in a command argument. Record the
registry digest printed by the push/inspect. Confirm the organization repository
exists and the account has push rights. This uses Docker Hub, not ECR.

## 2. Record the baseline — on the server

Connect using the supplied SSH key as `ubuntu`. Commands below run on that host.

```sh
sudo dokku domains:report civicsignal-tools
sudo dokku git:report civicsignal-tools
sudo dokku ps:report civicsignal-tools
sudo dokku nginx:show-config civicsignal-tools
sudo dokku certs:report civicsignal-tools
sudo dokku letsencrypt:cron-job
free -m
df -h /
sudo docker ps --format '{{.Names}} {{.Image}} {{.Status}}'
```

Save these non-secret outputs with the release record and check the portal/tool
URLs before making changes. Record any pre-existing failures. Do not export
environment variables or TLS private keys into the release notes.

## 3. Deploy a separate app for preview — on the server

Create a DNS-only A record `preview.civicsignal.africa` -> `34.250.120.147` in
Cloudflare after checking the hostname is unused. This record now exists; do not recreate it. Keep existing records unchanged.

```sh
sudo dokku apps:create civicsignal-web
sudo dokku domains:set civicsignal-web preview.civicsignal.africa
sudo dokku ports:set civicsignal-web http:80:8080
```

If the repository is private, use the installed Dokku app-specific registry
login. In a Bash session, read the password without echoing it:

```bash
read -r -s -p 'Docker Hub password/token: ' CS_REGISTRY_PASSWORD
printf '%s' "$CS_REGISTRY_PASSWORD" | sudo dokku registry:login --password-stdin civicsignal-web docker.io YOUR_DOCKER_HUB_USERNAME
unset CS_REGISTRY_PASSWORD
```

Existing registry authentication may already suffice; do not log out or change
global registry settings unnecessarily. Do not enable shell tracing.

Set the following value to the actual digest recorded in stage 1, not a guessed
digest or `latest`:

```sh
CS_IMAGE='codeforafrica/civicsignal-web@sha256:REPLACE_WITH_PUSHED_DIGEST'
sudo dokku git:from-image civicsignal-web "$CS_IMAGE"
sudo dokku ps:report civicsignal-web
sudo dokku ports:report civicsignal-web
curl --fail http://preview.civicsignal.africa/healthz
sudo dokku letsencrypt:set civicsignal-web email YOUR_OPERATIONS_EMAIL
sudo dokku letsencrypt:enable civicsignal-web
curl --fail https://preview.civicsignal.africa/healthz
```

Confirm the port mappings resolve to container port 8080, including HTTPS after
certificate installation. Use `ports:set civicsignal-web http:80:8080
https:443:8080` if needed. Preview all pages and login redirects over HTTPS. Check
the new app's logs and host resource use. Do not proceed if preview fails.

## 4. Transfer the public hostname — on the server

Schedule a brief cutover window: separate Dokku domain commands are not atomic.
Have the rollback commands below ready in a second SSH session.

Bootstrap apex HTTPS by importing the existing valid certificate into the NEW
app only. The old app keeps its own files and continues serving the portal.
The preview hostname temporarily loses certificate coverage during this step.

```sh
sudo dokku certs:add civicsignal-web /home/dokku/civicsignal-tools/tls/server.crt /home/dokku/civicsignal-tools/tls/server.key
sudo dokku ports:set civicsignal-web http:80:8080 https:443:8080
sudo dokku domains:remove civicsignal-tools civicsignal.africa
sudo dokku domains:add civicsignal-web civicsignal.africa www.civicsignal.africa
sudo nginx -t
sudo dokku letsencrypt:enable civicsignal-web --force
```

The forced issuance replaces the bootstrap certificate with one covering all
three new-app domains, including www and preview. Until it succeeds, www and
preview may report TLS errors. If issuance fails, execute rollback; do not leave
partially covered domains serving production. Never revoke the borrowed
certificate: the portal still uses its own copy.

Check certificate SANs and expiration, `letsencrypt:active civicsignal-web`, and
the renewal cron status. If no renewal job exists, arrange one explicitly with
infrastructure; do not silently change global cron configuration. Keep the portal
certificate and renewal arrangement intact. This runbook serves the website on
www as well as apex; it does not install a canonical www redirect.

## 5. Verify the cutover

From an external machine, check apex, www and preview over HTTPS, `/healthz`,
all pages, real JSON loading, login handoff, registration and password-reset
bookmarks. Verify `#/...` query/fragment preservation in a browser: fragments are
never sent to Nginx. Verify the unchanged portal and tool URLs against baseline.
Inspect `sudo dokku logs civicsignal-web --num 100` and resource use. Avoid
recording reset tokens or user credentials in screenshots or logs.

## Rollback — on the server

Restore apex to the STILL-RUNNING old app; no database restore or image build is
needed. Run only removals corresponding to domain assignments that succeeded.

```sh
sudo dokku domains:remove civicsignal-web civicsignal.africa www.civicsignal.africa
sudo dokku domains:add civicsignal-tools civicsignal.africa
sudo nginx -t
curl --fail https://civicsignal.africa/
curl --fail https://tools.civicsignal.africa/
```

Validate the old portal and homepage in a browser. The old certificate already
covers apex and tools. This restores the verified baseline; www had no explicit
old-app mapping or certificate coverage before launch. If www must remain valid
during rollback, separately add it to the old app and issue a suitable certificate
as an explicit extension, not as an assumed baseline. Leave the new app on
preview for diagnosis; if resources cause trouble, stop only `civicsignal-web`.
Do not delete either image or app during rollback.

For a later website-only revision, push a new immutable tag, record its digest,
and run `git:from-image civicsignal-web` with that digest. Retain the last working
new-site digest for image rollback. Never deploy a new-site image to
`civicsignal-tools`.

## References

The installed command help was checked on the server. Additional background:
[Dokku deployment](https://dokku.com/docs/deployment/methods/git/) and
[Dokku Let's Encrypt](https://github.com/dokku/dokku-letsencrypt).
