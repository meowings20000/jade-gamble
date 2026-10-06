const { chromium } = require('playwright');
(async () => {
  const B = 'http://127.0.0.1:3002';
  const b = await chromium.launch();
  const ct = await b.newContext({ viewport: { width: 1280, height: 900 } });
  const p = await ct.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push('PAGEERROR: ' + e.message));
  p.on('console', m => { if (m.type() === 'error') errs.push('CONSOLE: ' + m.text().slice(0, 200)); });
  p.on('requestfailed', r => errs.push('REQFAIL: ' + r.url().slice(-40)));
  await ct.request.post(B + '/api/auth/mock', { data: { username: 'btn' + Date.now() } });
  const shop = await (await ct.request.get(B + '/api/shop')).json();
  let best = null;
  for (const g of shop.grades || []) for (const it of g.items || []) if (!best || it.price < best.price) best = it;
  await ct.request.post(B + '/api/shop/buy', { data: { stone_id: best.id } });
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2200);
  await p.click('nav button[data-view="warehouse"]', { force: true });
  await p.waitForTimeout(1800);
  console.log('倉庫按鈕:', JSON.stringify(await p.$$eval('#view-warehouse button', e => e.map(x => x.innerText.trim()))));
  // 嘗試打開磨石面板：先找磨石/加工/處理，再退回點石頭卡片
  for (const e of await p.$$('#view-warehouse button')) {
    const t = (await e.innerText()).trim();
    if (/磨|加工|處理|切/.test(t)) { await e.click({ force: true }); break; }
  }
  await p.waitForTimeout(1500);
  let modal = await p.$$eval('.modal button', e => e.map(x => x.innerText.trim() + '|' + (x.dataset.force || '-'))).catch(() => null);
  if (!modal || !modal.length) {
    // 點第一張石頭卡片
    const card = await p.$('#view-warehouse .card, #view-warehouse .stone, #view-warehouse [data-stone]');
    if (card) await card.click({ force: true });
    await p.waitForTimeout(1500);
    modal = await p.$$eval('.modal button', e => e.map(x => x.innerText.trim() + '|' + (x.dataset.force || '-'))).catch(() => null);
  }
  console.log('磨石面板按鈕:', JSON.stringify(modal));
  // 依序點 正磨、重磨，看有沒有反應
  for (const want of ['正磨', '重磨']) {
    const before = errs.length;
    let hit = null;
    for (const e of await p.$$('.modal button')) { if ((await e.innerText()).includes(want)) { hit = e; break; } }
    if (!hit) { console.log(want, '：找不到按鈕'); continue; }
    await hit.click({ force: true });
    await p.waitForTimeout(3500);
    const t = await p.$eval('.modal', el => el.innerText.replace(/\s+/g, ' ').slice(0, 110)).catch(() => '(面板關了)');
    console.log(`按「${want}」→ 面板現在: ${t}${errs.length > before ? '  ⚠ ' + errs.slice(before).join(' | ') : ''}`);
    await p.evaluate(() => { const m = document.querySelector('.modal-bg'); if (m) m.remove(); });
    await p.click('nav button[data-view="warehouse"]', { force: true });
    await p.waitForTimeout(1200);
    for (const e of await p.$$('#view-warehouse button')) { const x = (await e.innerText()).trim(); if (/磨|加工|處理/.test(x)) { await e.click({ force: true }); break; } }
    await p.waitForTimeout(1500);
  }
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 5) : '無');
  await b.close();
})();
