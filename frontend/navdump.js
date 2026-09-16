const { chromium } = require('playwright');
const URL = process.env.JADE_URL || 'https://jade.meowmeow12245ouo.dpdns.org';
(async () => {
  const b = await chromium.launch();
  const p = await b.newPage({ viewport: { width: 1440, height: 900 } });
  await p.goto(URL, { waitUntil: 'networkidle' });
  const m = p.locator('#login-mock');
  if (await m.isVisible()) { await m.click(); await p.waitForTimeout(800); }
  console.log(JSON.stringify(await p.evaluate(() => [...document.querySelectorAll('nav button')].map(x => ({
    view: x.dataset.view, text: JSON.stringify(x.textContent), w: Math.round(x.getBoundingClientRect().width), display: getComputedStyle(x).display,
  }))), null, 1));
  console.log('lang:', await p.evaluate(() => localStorage.getItem('lang')));
  await b.close();
})();
