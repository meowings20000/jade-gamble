const { chromium } = require('playwright');
(async () => {
  const B = 'http://127.0.0.1:3002';
  const b = await chromium.launch();
  const ct = await b.newContext();
  await ct.request.post(B + '/api/auth/mock', { data: { username: 'hide' + Date.now() } });
  const shop = await (await ct.request.get(B + '/api/shop')).json();
  let best = null;
  for (const g of shop.grades || []) for (const it of g.items || []) if (!best || it.price < best.price) best = it;
  await ct.request.post(B + '/api/shop/buy', { data: { stone_id: best.id } });
  const p = await ct.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push(e.message));
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2000);
  // 直接叫出磨石面板（重磨＝故意選錯，看第一層保護）
  await p.evaluate((st) => startPolish(st, 3), { id: best.id, price: best.price });
  await p.waitForTimeout(1800);
  console.log('磨石面板:', (await p.$eval('.modal', el => el.innerText.replace(/\n+/g, ' | ').trim())).slice(0, 420));
  console.log('有沒有洩漏價值數字:', /目前價值|買入價|賺賠|落袋實拿/.test(await p.$eval('.modal', el => el.innerText)));
  // 磨一層，看爆裂率是否被壓到 25% 以下
  await p.click('#pol-adv', { force: true }).catch(() => {});
  await p.waitForTimeout(2000);
  const txt = await p.$eval('.modal', el => el.innerText.replace(/\n+/g, ' | ').trim()).catch(() => '(視窗關了)');
  console.log('磨一層後:', txt.slice(0, 300));
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 2) : '無');
  await b.close();
})();
