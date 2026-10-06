// nav_debug.js — 追為什麼登入後 #nav 還是 display:none
const { chromium } = require('playwright');

const URL = process.env.JADE_URL || 'https://jade.meowmeow12245ouo.dpdns.org';

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const logs = [];
  page.on('console', (m) => logs.push(`[${m.type()}] ${m.text()}`));
  page.on('pageerror', (e) => logs.push(`[pageerror] ${e.message}`));
  page.on('requestfailed', (r) => logs.push(`[reqfail] ${r.url()} ${r.failure() && r.failure().errorText}`));
  page.on('response', (r) => {
    const u = r.url();
    if (u.includes('/api/')) logs.push(`[http ${r.status()}] ${u.replace(URL, '')}`);
  });

  await page.goto(URL, { waitUntil: 'networkidle' });
  console.log('active view before login:', await page.evaluate(() => document.querySelector('.view.active')?.id));
  console.log('login-mock visible:', await page.locator('#login-mock').isVisible());

  const resp = await page.evaluate(async () => {
    const r = await fetch('/api/auth/mock', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: 'navdebug_' + Date.now() }),
    });
    const t = await r.text();
    const me = await fetch('/api/me');
    return { mockStatus: r.status, mockBody: t.slice(0, 200), meStatus: me.status, meBody: (await me.text()).slice(0, 300) };
  });
  console.log('mock login:', JSON.stringify(resp, null, 1));

  await page.reload({ waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);
  const after = await page.evaluate(() => ({
    activeView: document.querySelector('.view.active')?.id,
    navDisplay: getComputedStyle(document.querySelector('#nav')).display,
    navStyleAttr: document.querySelector('#nav').getAttribute('style'),
    userbox: document.querySelector('#userbox')?.textContent,
    chips: document.querySelector('#chips')?.textContent,
    headerHTML: document.querySelector('header').innerHTML.slice(0, 400),
  }));
  console.log('after reload:', JSON.stringify(after, null, 1));
  console.log('--- logs ---');
  console.log(logs.join('\n'));
  await page.screenshot({ path: 'nav-debug.png' });
  await browser.close();
})();
