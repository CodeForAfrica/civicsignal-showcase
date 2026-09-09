// Run with Node and Playwright installed; no requests modify the live portal.
const { chromium, request } = require('playwright');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const base = process.env.BASE_URL || 'http://127.0.0.1:8088';
const output = process.env.TEST_OUTPUT || '/tmp/civicsignal-web-tests';
const portal = 'https://tools.civicsignal.africa/';

(async () => {
  fs.mkdirSync(output, { recursive: true });
  const http = await request.newContext();
  const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_CHANNEL ? { channel: process.env.BROWSER_CHANNEL } : {}) });
  const report = { pages: [], blockers: [], failures: [], externalFailures: [] };
  try {
    assert.equal((await http.get(base + '/healthz')).status(), 200);
    for (const route of ['/missing-page', '/pipeline/content.db', '/.git/config', '/civicsignal.txt']) {
      assert.ok([403, 404].includes((await http.get(base + route)).status()), route);
    }
    const data = await http.get(base + '/showcase_data.json');
    if (!data.ok()) report.blockers.push('Missing showcase_data.json: Wavelength, Fingerprints and events map cannot render their data.');
    else await data.json();
    const checked = new Set();
    for (const file of fs.readdirSync(path.join(__dirname, '..')).filter(f => f.endsWith('.html'))) {
      const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
      const errors = [];
      page.on('pageerror', e => errors.push(e.message));
      page.on('requestfailed', r => {
        if (!r.url().startsWith(base)) report.externalFailures.push({ file, url: r.url(), error: r.failure()?.errorText });
      });
      const response = await page.goto(base + '/' + file, { waitUntil: 'networkidle', timeout: 60000 });
      assert.equal(response.status(), 200, file);
      assert.ok((await page.title()).length > 0, file + ' title');
      const refs = await page.locator('a[href], img[src], script[src], link[href]').evaluateAll(nodes => nodes.map(n => n.href || n.src));
      for (const ref of refs) {
        if (!ref.startsWith(base)) continue;
        const u = new URL(ref); u.hash = '';
        if (checked.has(u.href)) continue;
        checked.add(u.href);
        const r = await http.get(u.href);
        if (!r.ok()) report.failures.push(file + ': broken local reference ' + u.pathname + ' HTTP ' + r.status());
      }
      assert.equal(await page.locator('input[type=password]').count(), 0, file + ' collects passwords');
      assert.ok(await page.locator('a[href="' + portal + '#/login"]').count(), file + ' portal login');
      const brokenImages = await page.locator('img').evaluateAll(imgs => imgs.filter(i => !i.complete || !i.naturalWidth).map(i => i.getAttribute('src')));
      if (brokenImages.length) report.failures.push(file + ': broken images ' + brokenImages.join(', '));
      for (const width of [1440, 390]) {
        await page.setViewportSize({ width, height: 1000 });
        await page.waitForTimeout(150);
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1);
        if (overflow) report.failures.push(file + ': horizontal overflow at ' + width);
        if (['index.html', 'login.html', 'wavelength.html'].includes(file)) {
          await page.screenshot({ path: path.join(output, file + '-' + width + '.png'), fullPage: true });
        }
      }
      if (errors.length) report.failures.push(...errors.map(e => file + ': ' + e));
      report.pages.push({ file, status: response.status(), errors });
      await page.close();
    }
    // Intercept portal navigation to test exact forwarding without submitting anything.
    const page = await browser.newPage();
    await page.route(portal + '**', route => route.fulfill({ status: 200, body: '<title>Portal navigation test</title>' }));
    for (const suffix of ['#/login', '#/user/signup', '#/user/reset-password?email=test%40example.com&password_reset_token=test-only', '?source=bookmark#/user/activated', '#/privacy']) {
      await page.goto(base + '/' + suffix);
      await page.waitForURL(portal + suffix);
      assert.equal(page.url(), portal + suffix);
    }
    await page.goto(base + '/#mediacloud');
    assert.equal(new URL(page.url()).origin, base);
    await page.evaluate(() => { location.hash = '/login'; });
    await page.waitForURL(portal + '#/login');
    await page.goto(base + '/');
    await page.locator('a.cs-login-btn').click();
    await page.waitForURL(portal + '#/login');
    report.legacyRedirects = 'passed: login, signup, reset parameters, activation, privacy, hash changes and ordinary anchors';
    report.localReferencesChecked = checked.size;
  } finally {
    fs.writeFileSync(path.join(output, 'report.json'), JSON.stringify(report, null, 2));
    console.log(JSON.stringify(report, null, 2));
    await browser.close();
    await http.dispose();
  }
  process.exitCode = report.failures.length ? 1 : report.blockers.length ? 2 : 0;
})().catch(e => { console.error(e); process.exitCode = 1; });
