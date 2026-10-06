const { chromium } = require('playwright');
const B = 'http://127.0.0.1:3002';
(async () => {

  async function buy(ctx) {
    const shop = await (await ctx.get(B + '/api/shop')).json();
    let best = null;
    for (const g of shop.grades || []) for (const it of g.items || []) if (!best || it.price < best.price) best = it;
    await ctx.post(B + '/api/shop/buy', { data: { stone_id: best.id } });
    return best;
  }
  const b = await chromium.launch();
  const ct = await b.newContext({ viewport: { width: 1280, height: 900 } });
  const p = await ct.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push('PAGEERROR: ' + e.message));
  await ct.request.post(B + '/api/auth/mock', { data: { username: 'fx' + Date.now() } });
  const A = await buy(ct.request);
  await ct.request.post(B + '/api/polish/start', { data: { stone_id: A.id, force: 1 } }); // A 已在磨
  const Bst = await buy(ct.request);
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2000);

  // 已在磨的石頭：按磨石應該直接續磨
  await p.evaluate(() => { const m = document.querySelector('.modal-bg'); if (m) m.remove(); });
  await p.evaluate((st) => doPolish(st), A);
  await p.waitForTimeout(2000);
  console.log('A（已在磨）→', (await p.$eval('.modal', el => el.innerText.replace(/\s+/g, ' ').slice(0, 80)).catch(() => '(沒面板)')));

  // 沒開磨的石頭：應該看到選力度
  await p.evaluate(() => { const m = document.querySelector('.modal-bg'); if (m) m.remove(); });
  await p.evaluate((st) => doPolish(st), Bst);
  await p.waitForTimeout(1500);
  const picker = await p.$$eval('.modal button', e => e.map(x => x.innerText.split('\n')[0].trim())).catch(() => []);
  console.log('B（未開磨）面板按鈕:', JSON.stringify(picker));
  let hit = null;
  for (const e of await p.$$('.modal button')) { if ((await e.innerText()).includes('正磨')) { hit = e; break; } }
  if (hit) {
    await hit.click({ force: true });
    await p.waitForTimeout(2500);
    console.log('B 按「正磨」→', (await p.$eval('.modal', el => el.innerText.replace(/\s+/g, ' ').slice(0, 80)).catch(() => '(沒面板)')));
  }
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 4) : '無');
  await b.close();
})();
