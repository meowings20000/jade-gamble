
const { chromium } = require('playwright');
(async () => {
  const B = 'http://127.0.0.1:3002';
  const b = await chromium.launch();
  const ctx = await b.newContext({ viewport: { width: 420, height: 900 } });
  const p = await ctx.newPage();
  const errs = [];
  p.on('pageerror', (e) => errs.push(String(e).slice(0, 120)));
  await p.goto(B + '/');
  await ctx.request.post(B + '/api/auth/mock', { data: { username: 'ui' + Date.now() } });
  await p.goto(B + '/');
  await p.waitForTimeout(1800);
  // 點奪寶
  await p.click('nav button[data-view="heist"]');
  await p.waitForTimeout(1200);
  const lobby = await p.$eval('#heist-body', (el) => el.innerText.replace(/\s+/g, ' ').slice(0, 260)).catch(() => 'NO #heist-body');
  const joinBtns = await p.$$eval('#heist-body button', (bs) => bs.length);
  console.log('【大廳】', lobby);
  console.log('【大廳按鈕數】', joinBtns);
  // 入場（公斤料）
  await p.click('#heist-body button');
  await p.waitForTimeout(2000);
  const table = await p.$eval('#heist-body', (el) => el.innerText.replace(/\s+/g, ' ').slice(0, 300)).catch(() => 'NO');
  console.log('【入場後】', table);
  // 等 bot 補位
  await p.waitForTimeout(22000);
  const after = await p.$eval('#heist-body', (el) => el.innerText.replace(/\s+/g, ' ').slice(0, 320)).catch(() => 'NO');
  console.log('【補 bot 後】', after);
  // 出手：合作
  const coop = await p.$('#heist-body button:has-text("合作")');
  console.log('【有合作按鈕】', !!coop);
  if (coop) {
    await coop.click();
    await p.waitForTimeout(2500);
    const res = await p.$eval('#heist-body', (el) => el.innerText.replace(/\s+/g, ' ').slice(0, 320)).catch(() => 'NO');
    console.log('【出手後】', res);
  }
  console.log('【JS 錯誤】', errs.length ? errs : '無');
  await b.close();
})();
