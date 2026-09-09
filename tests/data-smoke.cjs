const { chromium } = require('playwright');
const assert = require('node:assert/strict');
const base = process.env.BASE_URL || 'http://127.0.0.1:8088';
(async () => {
  const browser = await chromium.launch({ headless: true, ...(process.env.BROWSER_CHANNEL ? { channel: process.env.BROWSER_CHANNEL } : {}) });
  const page = await browser.newPage();
  try {
    await page.goto(base + '/fingerprints.html', { waitUntil: 'networkidle' });
    assert.ok(await page.locator('#cardCarousels .card').count() > 0);
    const before = await page.locator('#pubCount').innerText();
    const country = await page.locator('#fCountry option').nth(1).getAttribute('value');
    await page.selectOption('#fCountry', country);
    await page.waitForTimeout(300);
    assert.notEqual(await page.locator('#pubCount').innerText(), before);
    console.log('Fingerprints: populated cards and working country filter');
    await page.goto(base + '/wavelength.html', { waitUntil: 'networkidle' });
    assert.ok(await page.evaluate(() => {
      const chart = Chart.getChart('streamChart');
      return chart && chart.data.datasets.length > 0;
    }));
    console.log('Wavelength: populated stream chart');
    await page.goto(base + '/major-events-map.html', { waitUntil: 'networkidle' });
    assert.ok(await page.locator('#mainContent').isVisible());
    assert.ok(await page.locator('#map .leaflet-interactive').count() > 0);
    console.log('Events map: rendered interactive shapes');
  } finally {
    await browser.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
