
const { chromium, devices } = require('playwright');
(async () => {
  const B='http://127.0.0.1:3002';
  const b=await chromium.launch();
  const ctx=await b.newContext({ ...devices['iPhone 12'] });
  const p=await ctx.newPage();
  p.on('pageerror', e=>console.log('JS ERROR:', e.message));
  await ctx.request.post(B+'/api/auth/mock',{data:{username:'mob2'+Date.now()}});
  await p.goto(B+'/',{waitUntil:'domcontentloaded'}); await p.waitForTimeout(1800);
  await p.click('nav button[data-view="heist"]'); await p.waitForTimeout(1500);
  await (await p.$('#heist-body button')).click();               // 入場
  await p.waitForTimeout(1800);
  for (const e of await p.$$('#heist-body button')) { const t=await e.innerText(); if (t.includes('直接開局')) { await e.click(); break; } }
  await p.waitForTimeout(2500);                                   // 開局中
  const info = await p.evaluate(() => {
    const vw = innerWidth;
    const btns = [...document.querySelectorAll('#heist-body button')].map(e => { const r=e.getBoundingClientRect();
      return { t: e.innerText.trim().slice(0,16), w: Math.round(r.width), inside: r.right <= vw+1 && r.left >= -1 }; });
    return { vw, sw: document.documentElement.scrollWidth, btns, seats: document.querySelectorAll('#heist-body .seat').length };
  });
  console.log('【手機 390px 進行中】內容寬', info.sw, '| 橫向溢出:', info.sw > info.vw+1, '| 座位數', info.seats);
  for (const x of info.btns) console.log('   ', x.t.padEnd(18), x.w+'px', x.inside?'OK':'超出 ✗');
  console.log('全部按鈕可見:', info.btns.every(x=>x.inside), '| 按鈕數:', info.btns.length);
  await p.screenshot({ path: 'mobile_heist.png' });
  await b.close();
})();
