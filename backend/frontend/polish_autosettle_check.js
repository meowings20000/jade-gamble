const { chromium } = require('playwright');
const B = 'http://127.0.0.1:3002';
(async () => {
  const b = await chromium.launch();
  const ct = await b.newContext({ viewport: { width: 1280, height: 900 } });
  const p = await ct.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push('PAGEERROR: ' + e.message));
  await ct.request.post(B + '/api/auth/mock', { data: { username: 'auto' + Date.now() } });
  const buy = async () => {
    const shop = await (await ct.request.get(B + '/api/shop')).json();
    let best = null;
    for (const g of shop.grades || []) for (const it of g.items || []) if (!best || it.price < best.price) best = it;
    await ct.request.post(B + '/api/shop/buy', { data: { stone_id: best.id } });
    return best;
  };
  const me = async () => await (await ct.request.get(B + '/api/me')).json();
  const A = await buy();
  const Bst = await buy();
  await ct.request.post(B + '/api/polish/start', { data: { stone_id: A.id, force: 2 } });
  const m0 = await me();
  console.log('A 在磨 → polish_running=%s｜喵喵幣 %s', m0.polish_running, m0.chips);

  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2000);
  // 直接在頁面上磨 B（後端應該自動結算 A）
  await p.evaluate((st) => startPolish(st, 1), Bst);
  await p.waitForTimeout(3000);
  const m1 = await me();
  console.log('磨 B 之後 → polish_running=%s｜喵喵幣 %s（%+d）', m1.polish_running, m1.chips, m1.chips - m0.chips);
  console.log('面板:', (await p.$eval('.modal', el => el.innerText.replace(/\s+/g, ' ').slice(0, 70)).catch(() => '(沒面板)')));
  console.log('有沒有卡人的「還有一顆在磨」:', (await p.$$eval('.modal', e => e.map(x => x.innerText.includes('還有一顆在磨')))).some(Boolean));
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 3) : '無');
  await b.close();
})();
