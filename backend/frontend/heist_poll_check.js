
const { chromium } = require('playwright');
(async () => {
  const B='http://127.0.0.1:3002';
  const b=await chromium.launch(); const ctx=await b.newContext(); const p=await ctx.newPage();
  let n=0; p.on('request', r => { if (r.url().includes('/api/heist')) n++; });
  p.on('pageerror', e=>console.log('JS ERROR:', e.message));
  await ctx.request.post(B+'/api/auth/mock',{data:{username:'pollchk'+Date.now()}});
  await p.goto(B+'/',{waitUntil:'domcontentloaded'}); await p.waitForTimeout(1500);
  await p.click('nav button[data-view="heist"]'); await p.waitForTimeout(3000);
  const a=n; console.log('/api/heist 請求數（進入大廳 3 秒後）:', a);
  await p.waitForTimeout(8000);
  console.log('再等 8 秒後總數:', n, '| 8 秒內新增:', n-a, '（每 2.5 秒一次應約 3 次）');
  await b.close();
})();
