const { chromium } = require('playwright');
(async () => {
  const B = 'http://127.0.0.1:3002';
  // 1) API：確認落袋回應帶了 polish / stone_price / net
  const c = await require('playwright').request.newContext();
  await c.post(B + '/api/auth/mock', { data: { username: 'net' + Date.now() } });
  const shop = await (await c.get(B + '/api/shop')).json();
  let best = null;
  for (const g of shop.grades || []) for (const it of g.items || []) if (!best || it.price < best.price) best = it;
  await c.post(B + '/api/shop/buy', { data: { stone_id: best.id } });
  const st = await (await c.post(B + '/api/polish/start', { data: { stone_id: best.id, force: 2 } })).json();
  const cash = await (await c.post(B + '/api/polish/cash', { data: { stone_id: best.id } })).json();
  console.log('API 落袋回應: 價格=%s payout=%s net=%s polish=%s' , cash.stone_price, cash.payout, cash.net, cash.polish);

  // 2) 前端：用真實形狀的 payload 渲染結果視窗，看文字
  const b = await chromium.launch();
  const ct = await b.newContext();
  await ct.request.post(B + '/api/auth/mock', { data: { username: 'ui' + Date.now() } });
  const p = await ct.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push(e.message));
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2000);

  // 賺：成本 100、落袋 127
  await p.evaluate(() => showResultModal('磨石落袋', { payout: 127, multiplier: 1.3392, quality: '砖头料', variety: '底色', polish: true, stone_price: 100, net: 27 }, null));
  await p.waitForTimeout(600);
  const winText = await p.$eval('.modal', el => el.innerText.replace(/\n+/g, ' | ').trim());
  const winCls = await p.$eval('.modal .big-result', el => el.className).catch(() => '(無)');
  console.log('賺的案例 →', winText);
  console.log('   大數字 class:', winCls, '（不該是 lose）');
  await p.evaluate(() => document.querySelector('.modal-bg').remove());

  // 賠：成本 1,000、落袋 127
  await p.evaluate(() => showResultModal('磨石落袋', { payout: 127, multiplier: 0.93, quality: '砖头料', variety: '底色', polish: true, stone_price: 1000, net: -873 }, null));
  await p.waitForTimeout(600);
  console.log('賠的案例 →', await p.$eval('.modal', el => el.innerText.replace(/\n+/g, ' | ').trim()));
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 3) : '無');
  await b.close();
})();
