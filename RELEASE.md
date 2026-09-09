# Production release — 9 September 2026

The rendered website is live at https://civicsignal.africa and
https://www.civicsignal.africa. Preview: https://preview.civicsignal.africa.

## Current automated release

The first [deploy_to_prod run](https://github.com/CodeForAfrica/civicsignal-showcase/actions/runs/34388337811)
succeeded on 9 September 2026. It deployed commit `096c717` as
`codeforafrica/civicsignal-web@sha256:de578ae15fefccc6d8520b254194cbea41aa9a2ed6aa28d2e066082a0fbdc7c1`
through AWS OIDC, SSM and Dokku. Setup and verification details are in
[deploy/SETUP.md](deploy/SETUP.md). Routine releases now use that manual-dispatch
GitHub Action. The following sections record the initial manual cutover.

## Initial manual artifact and host

- Rendered source and container implementation: commit `b3d49a9`.
- Docker Hub tag: `codeforafrica/civicsignal-web:2026-09-09-02`.
- Deployed registry digest: `sha256:2a8d13c22b5ad352a32c721d0315b323ec3c74dfc563bb1f9aab4a8545974139`.
- Architecture: linux/arm64. Dataset: 9 September 2026.
- Dokku app: `civicsignal-web`, on `34.250.120.147` in AWS eu-west-1.
- Port mappings: `http:80:8080 https:443:8080`.
- Container: `civicsignal-web.web.1`; Docker health status healthy.

Docker Hub publishing and Dokku `git:from-image` were executed manually.
At that initial cutover, no automated release pipeline had been added.

## DNS and TLS

A DNS-only A record was created for preview at the existing server IP. Apex and
www DNS were unchanged. Apex was moved from `civicsignal-tools` to the new app;
www was explicitly assigned to the new app. All three new-app hostnames have a
Let's Encrypt certificate expiring 8 December 2026. The existing renewal cron
was already enabled and was left intact. Operations contact was inherited as
support@codeforafrica.org.

## Verification

- Local, HTTPS preview and public-site smoke suites passed: 14 HTML pages,
  35 local references, no broken images or uncaught JavaScript exceptions,
  no horizontal page overflow at desktop/mobile test widths.
- Login button and legacy fragment forwarding passed, including signup,
  reset-password parameters and ordinary anchor preservation.
- Published data checks passed: Fingerprints cards and country filter,
  Wavelength stream chart, and event-map interactive shapes.
- Main, www and preview health endpoints returned HTTP 200 over valid HTTPS.
- Portal, Explorer, Sources and Topics HTML responses were byte-for-byte
  unchanged compared with the pre-cutover baseline.
- All existing portal/tool/database container IDs were unchanged; no existing
  container was restarted or redeployed as part of this release.
- Real account authentication was not exercised. Redirect checks and unchanged
  portal responses do not establish a successful authenticated session.

Browser reports from execution are in `/tmp/civicsignal-preview-tests` and
`/tmp/civicsignal-production-tests` on the operator workstation. Future checks:

```sh
BASE_URL=https://civicsignal.africa BROWSER_CHANNEL=chrome node tests/local-smoke.cjs
BASE_URL=https://civicsignal.africa BROWSER_CHANNEL=chrome node tests/data-smoke.cjs
```

## Rollback target

The old `civicsignal-tools` app is still running, serving
https://tools.civicsignal.africa. Its original image is
`codeforafrica/civicsignal-web-tools:3.20.3`. Its existing TLS certificate still
covers tools and apex. Restore apex routing to that app using the rollback
section of [MANUAL_DEPLOY.md](MANUAL_DEPLOY.md); do not rebuild or delete it.
Rollback was prepared but not exercised against production.
