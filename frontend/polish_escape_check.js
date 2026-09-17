const { chromium } = require('playwright');
const B = 'http://127.0.0.1:3002';
(async () => {
  const b = await chromium.launch();
  const ct = await b.newContext({ viewport: { width: 1280, height: 900 } });
  const p = await ct.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push('PAGEERROR: ' + e.message));
  await ct.request.post(B + '/api/auth/mock', { data: { username: 'esc' + Date.now() } });
  const buy = async () => {
    const shop = await (await ct.request.get(B + '/api/shop')).json();
    let best = null;
    for (const g of shop.grades || []) for (const it of g.items || []) if (!best || it.price < best.price) best = it;
    await ct.request.post(B + '/api/shop/buy', { data: { stone_id: best.id } });
    return best;
  };
  const A = await buy();
  const Bst = await buy();
  await ct.request.post(B + '/api/polish/start', { data: { stone_id: A.id, force: 2 } });
  const meBefore = await (await ct.request.get(B + '/api/me')).json();
  console.log('A 在磨｜/api/me: polish_running=%s polish_stone=%s（A.id=%s）',
    meBefore.polish_running, meBefore.polish_stone, A.id);
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2000);

  // 按別顆（B）→ 應該出現「還有一顆在磨」+「結算那一顆」
  await p.evaluate((st) => doPolish(st), Bst);
  await p.waitForTimeout(1500);
  console.log('按 B →', (await p.$eval('.modal', el => el.innerText.replace(/\s+/g, ' ').slice(0, 60)).catch(() => '(沒面板)')));
  console.log('   按鈕:', JSON.stringify(await p.$$eval('.modal button', e => e.map(x => x.innerText.trim())).catch(() => [])));

  // 按「結算那一顆」
  let hit = null;
  for (const e of await p.$$('.modal button')) { if ((await e.innerText()).includes('結算那一顆')) { hit = e; break; } }
  const chipsBefore = (await (await ct.request.get(B + '/api/me')).json()).chips;
  if (hit) { await hit.click({ force: true }); await p.waitForTimeout(3000); }
  const after = await (await ct.request.get(B + '/api/me')).json();
  console.log('結算後: polish_running=%s｜喵喵幣 %s → %s', after.polish_running, chipsBefore, after.chips);
  console.log('接下來畫面:', (await p.$eval('.modal', el => el.innerText.replace(/\s+/g, ' ').slice(0, 60)).catch(() => '(沒面板)')));
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 3) : '無');
  await b.close();
})();
