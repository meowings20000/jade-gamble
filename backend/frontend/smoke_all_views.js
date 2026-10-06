const { chromium } = require('playwright');
(async () => {
  const B = 'http://127.0.0.1:3002';
  const b = await chromium.launch();
  const ct = await b.newContext({ viewport: { width: 1280, height: 900 } });
  const p = await ct.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push('PAGEERROR: ' + e.message));
  p.on('console', m => { if (m.type() === 'error') errs.push('CONSOLE: ' + m.text().slice(0, 120)); });
  await ct.request.post(B + '/api/auth/mock', { data: { username: 'smoke' + Date.now() } });
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2500);
  const views = ['shop', 'warehouse', 'market', 'transfer', 'history', 'bank', 'heist', 'exchange', 'collection', 'ranks'];
  for (const v of views) {
    const before = errs.length;
    const ok = await p.click(`nav button[data-view="${v}"]`, { force: true, timeout: 6000 }).then(() => true).catch(() => false);
    await p.waitForTimeout(1800);
    const txt = await p.$eval(`#view-${v}`, el => el.innerText.replace(/\s+/g, ' ').trim()).catch(() => '');
    console.log(`${v.padEnd(11)} 點得到=${ok} 內容長度=${txt.length}${errs.length > before ? '  ⚠ ' + errs.slice(before).join(' | ') : ''}`);
  }
  console.log('總共 JS 錯誤:', errs.length ? errs : '無');
  await b.close();
})();
