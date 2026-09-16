
const { chromium } = require('playwright');
(async () => {
  const B = 'http://127.0.0.1:3002';
  const b = await chromium.launch();
  const ctx = await b.newContext();
  const p = await ctx.newPage();
  await ctx.request.post(B + '/api/auth/mock', { data: { username: 'cdchk' + Date.now() } });
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(1800);
  await p.click('nav button[data-view="heist"]');
  await p.waitForTimeout(1200);
  const join = await p.$('#heist-body button');
  await join.click();
  await p.waitForTimeout(2500);
  const cd1 = await p.$eval('#heist-cd', el => el.textContent).catch(() => '(沒有倒數)');
  console.log('等待中倒數 t=0   :', cd1);
  await p.waitForTimeout(4000);
  const cd2 = await p.$eval('#heist-cd', el => el.textContent).catch(() => '(沒有倒數)');
  console.log('等待中倒數 t=+4s :', cd2, '| 有在跑:', cd1 !== cd2);
  await b.close();
})();
