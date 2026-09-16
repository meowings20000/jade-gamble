
const { chromium, devices } = require('playwright');
(async () => {
  const B='http://127.0.0.1:3002';
  const b=await chromium.launch();
  const ctx=await b.newContext({ ...devices['iPhone 12'] });
  const p=await ctx.newPage();
  p.on('pageerror', e=>console.log('JS ERROR:', e.message));
  await ctx.request.post(B+'/api/auth/mock',{data:{username:'mob'+Date.now()}});
  await p.goto(B+'/',{waitUntil:'domcontentloaded'}); await p.waitForTimeout(1800);
  await p.click('nav button[data-view="heist"]'); await p.waitForTimeout(1500);
  const before = await p.evaluate(() => ({ w: innerWidth, sw: document.documentElement.scrollWidth }));
  console.log('大廳：視窗寬', before.w, '| 內容寬', before.sw, '| 橫向溢出:', before.sw > before.w + 1);
  await (await p.$('#heist-body button')).click();
  await p.waitForTimeout(2500);
  const info = await p.evaluate(() => {
    const vw = innerWidth;
    const btns = [...document.querySelectorAll('#heist-body button')].map(e => ({
      t: e.innerText.trim().slice(0,14),
      w: Math.round(e.getBoundingClientRect().width),
      right: Math.round(e.getBoundingClientRect().right),
      inside: e.getBoundingClientRect().right <= vw + 1 && e.getBoundingClientRect().left >= -1
    }));
    return { vw, sw: document.documentElement.scrollWidth, btns };
  });
  console.log('桌內：視窗寬', info.vw, '| 內容寬', info.sw, '| 橫向溢出:', info.sw > info.vw + 1);
  console.log('按鈕（寬度 / 是否都在畫面內）:');
  for (const x of info.btns) console.log('   ', x.t.padEnd(16), x.w + 'px', x.inside ? 'OK' : '超出畫面 ✗');
  console.log('全部按鈕都看得到:', info.btns.every(x => x.inside));
  await p.screenshot({ path: 'mobile_heist.png', fullPage: true });
  console.log('截圖: mobile_heist.png');
  await b.close();
})();
