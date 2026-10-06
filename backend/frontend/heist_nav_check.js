
const { chromium } = require('playwright');
(async () => {
  const B = 'http://127.0.0.1:3002';
  const b = await chromium.launch();
  const ctx = await b.newContext();
  const p = await ctx.newPage();
  p.on('pageerror', e => console.log('JS ERROR:', e.message));
  await ctx.request.post(B + '/api/auth/mock', { data: { username: 'navchk' + Date.now() } });
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2000);
  const active = async () => await p.$eval('div.view.active', el => el.id).catch(() => '(none)');
  console.log('登入後視圖:', await active());
  await p.click('nav button[data-view="heist"]');
  await p.waitForTimeout(1500);
  console.log('點奪寶後:', await active());
  const joinBtn = await p.$('#heist-body button');
  if (!joinBtn) { console.log('找不到入場鈕'); await b.close(); return; }
  console.log('入場鈕文字:', (await joinBtn.innerText()).trim());
  await joinBtn.click();
  await p.waitForTimeout(2500);
  console.log('按入場後視圖:', await active(), '  ← 應該還是 view-heist');
  const txt = await p.$eval('#heist-body', el => el.innerText.slice(0, 260)).catch(() => '');
  console.log('畫面內容:', txt.replace(/\n+/g, ' | ').slice(0, 230));
  console.log('有退出排隊鈕:', txt.includes('退出排隊'));
  await b.close();
})();
