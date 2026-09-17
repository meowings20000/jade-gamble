const { chromium } = require('playwright');
const B = 'http://127.0.0.1:3002';
(async () => {
  // 1) 伺服器：三種力度各試一次
  const c = await require('playwright').request.newContext();
  await c.post(B + '/api/auth/mock', { data: { username: 'p3' + Date.now() } });
  for (const force of [1, 2, 3]) {
    const shop = await (await c.get(B + '/api/shop')).json();
    let best = null;
    for (const g of shop.grades || []) for (const it of g.items || []) if (!best || it.price < best.price) best = it;
    await c.post(B + '/api/shop/buy', { data: { stone_id: best.id } });
    const r = await c.post(B + '/api/polish/start', { data: { stone_id: best.id, force } });
    const j = await r.json();
    console.log(`API force=${force} → HTTP ${r.status()}`, JSON.stringify(j).slice(0, 130));
  }

  // 2) 前端：直接開磨石面板，依序點 正磨 / 重磨 / 離開
  const b = await chromium.launch();
  const ct = await b.newContext({ viewport: { width: 1280, height: 900 } });
  const p = await ct.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push('PAGEERROR: ' + e.message));
  await ct.request.post(B + '/api/auth/mock', { data: { username: 'p3ui' + Date.now() } });
  const shop = await (await ct.request.get(B + '/api/shop')).json();
  let best = null;
  for (const g of shop.grades || []) for (const it of g.items || []) if (!best || it.price < best.price) best = it;
  await ct.request.post(B + '/api/shop/buy', { data: { stone_id: best.id } });
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2200);
  await p.evaluate((st) => doPolish(st), best);
  await p.waitForTimeout(1200);
  const btns = await p.$$eval('.modal button', e => e.map(x => x.innerText.split('\n')[0].trim() + '|' + (x.dataset.force || '-')));
  console.log('面板按鈕:', JSON.stringify(btns));
  for (const want of ['正磨', '重磨', '離開']) {
    let hit = null;
    for (const e of await p.$$('.modal button')) { if ((await e.innerText()).includes(want)) { hit = e; break; } }
    if (!hit) { console.log(want, '找不到'); continue; }
    const before = errs.length;
    await hit.click({ force: true });
    await p.waitForTimeout(3200);
    const m = await p.$eval('.modal', el => el.innerText.replace(/\s+/g, ' ').slice(0, 90)).catch(() => '(沒有面板了)');
    console.log(`點「${want}」→ ${m}${errs.length > before ? '  ⚠ ' + errs.slice(before).join(' | ') : ''}`);
    // 重開面板繼續測下一個
    await p.evaluate((st) => doPolish(st), best);
    await p.waitForTimeout(1000);
  }
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 4) : '無');
  await b.close();
})();
