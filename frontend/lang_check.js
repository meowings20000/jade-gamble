const { chromium } = require('playwright');
const URL = process.env.JADE_URL || 'https://jade.meowmeow12245ouo.dpdns.org';
(async () => {
  const b = await chromium.launch();
  const p = await b.newPage({ viewport: { width: 1280, height: 900 } });
  const errs = [];
  p.on('pageerror', e => errs.push('pageerror: ' + e.message));
  p.on('console', m => { if (m.type() === 'error') errs.push('console: ' + m.text()); });
  await p.goto(URL, { waitUntil: 'networkidle' });
  const m = p.locator('#login-mock');
  if (await m.isVisible()) { await m.click(); await p.waitForTimeout(900); }
  const dump = async (tag) => {
    const d = await p.evaluate(() => ({
      lang: localStorage.getItem('lang'),
      toggleText: document.querySelector('#lang-toggle')?.textContent,
      nav: [...document.querySelectorAll('nav button[data-view]')].map(x => x.textContent),
      shopH2: document.querySelector('#view-shop h2')?.textContent,
      modeBtns: [document.querySelector('#classic-bet')?.textContent, document.querySelector('#yboss-bet')?.textContent],
      logout: document.querySelector('#logout')?.textContent,
    }));
    console.log(tag, JSON.stringify(d));
  };
  await dump('zh-TW:');
  await p.click('#lang-toggle');
  await p.waitForTimeout(600);
  await dump('切換後:');
  await p.click('#lang-toggle');
  await p.waitForTimeout(600);
  await dump('切回:');
  console.log('errors:', errs.slice(0, 6));
  await b.close();
})();
