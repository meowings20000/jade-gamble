const { chromium } = require('playwright');
(async () => {
  const b = await chromium.launch();
  const p = await b.newPage({ viewport: { width: 1100, height: 900 } });
  await p.goto('http://127.0.0.1:3002/', { waitUntil: 'networkidle' });
  const m = p.locator('#login-mock');
  if (await m.isVisible()) { await m.click(); await p.waitForTimeout(800); }
  await p.click('nav button[data-view="bank"]');
  await p.waitForTimeout(1200);
  const d = await p.evaluate(() => ({
    offer: getComputedStyle(document.querySelector('#bank-offer')).display,
    appeal: getComputedStyle(document.querySelector('#bank-appeal-card')).display,
    terms: document.querySelector('#bank-terms').textContent.slice(0, 60),
    current: document.querySelector('#bank-current').textContent.slice(0, 30),
  }));
  console.log(JSON.stringify(d, null, 1));
  await p.screenshot({ path: 'bank-ui.png' });
  await b.close();
})();
