
const { chromium } = require('playwright');
(async () => {
  const B='http://127.0.0.1:3002';
  const b=await chromium.launch(); const ctx=await b.newContext(); const p=await ctx.newPage();
  p.on('pageerror', e=>console.log('JS ERROR:', e.message));
  await ctx.request.post(B+'/api/auth/mock',{data:{username:'btnchk'+Date.now()}});
  await p.goto(B+'/',{waitUntil:'domcontentloaded'}); await p.waitForTimeout(1800);
  await p.click('nav button[data-view="heist"]'); await p.waitForTimeout(1500);
  await (await p.$('#heist-body button')).click();  // 入場
  await p.waitForTimeout(2500);
  const btns = await p.$$eval('#heist-body button', els => els.map(e => e.innerText.trim()));
  console.log('等待中畫面上的按鈕:', JSON.stringify(btns, null, 0));
  console.log('有補 bot 鈕:', btns.some(t => t.includes('直接開局')));
  console.log('有退出鈕  :', btns.some(t => t.includes('退出排隊')));
  // 按下補 bot，應該開局
  for (const e of await p.$$('#heist-body button')) { const t = await e.innerText(); if (t.includes('直接開局')) { await e.click(); break; } }
  await p.waitForTimeout(2500);
  const body = await p.$eval('#heist-body', el => el.innerText.replace(/\n+/g,' | ').slice(0,150)).catch(()=>'');
  console.log('按補 bot 後:', body);
  await b.close();
})();
