# Rendered CivicSignal website

Deployment source: CodeForAfrica/civicsignal-showcase. The build/pipeline source
is CodeForAfrica/civicsignal-web. This repository includes the published HTML,
assets and showcase_data.json (snapshot dated 9 September 2026).

The ARM64 Nginx image serves only root HTML, assets and the JSON on port 8080.
Login links lead to the existing portal. Legacy #/... bookmarks forward there
with their parameters intact. No backend services or credentials are included.

```sh
docker build --platform linux/arm64 -t civicsignal-web:local -t codeforafrica/civicsignal-web:2026-09-09-02 .
docker run -d --name civicsignal-web-local -p 127.0.0.1:8088:8080 civicsignal-web:local
```

Stop and remove the existing local container before recreating it. Open
http://localhost:8088. Health endpoint: /healthz.

With Node and Playwright installed:

```sh
BROWSER_CHANNEL=chrome node tests/local-smoke.cjs
```

This uses installed Chrome; omit BROWSER_CHANNEL for Playwright Chromium.
TEST_OUTPUT controls report/screenshot destination (default /tmp/civicsignal-web-tests).

See [MANUAL_DEPLOY.md](MANUAL_DEPLOY.md) for image publishing, preview, domain
cutover, HTTPS and rollback. No production deployment has been performed.

Upstream rendered HTML refreshes may overwrite local portal/link fixes: preserve
those changes when merging and rerun browser tests before publishing each image.
