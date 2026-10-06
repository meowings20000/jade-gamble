
const { chromium } = require('playwright');
(async () => {
  const B='http://127.0.0.1:3002';
  const b=await chromium.launch(); const ctx=await b.newContext(); const p=await ctx.newPage();
  p.on('pageerror', e=>console.log('JS ERROR:', e.message));
  await ctx.request.post(B+'/api/auth/mock',{data:{username:'rndchk'+Date.now()}});
  await p.goto(B+'/',{waitUntil:'domcontentloaded'}); await p.waitForTimeout(1800);
  await p.click('nav button[data-view="heist"]'); await p.waitForTimeout(1200);
  await (await p.$('#heist-body button')).click(); await p.waitForTimeout(1200);
  const fill = await p.$$('#heist-body button');
  for (const btn of fill) { const t=await btn.innerText(); if (t.includes('直接開局')) { await btn.click(); break; } }
  await p.waitForTimeout(2000);
  const active = async () => await p.$eval('div.view.active', el=>el.id).catch(()=>'(none)');
  console.log('開局後視圖:', await active());
  for (let i=1;i<=3;i++){
    const btns = await p.$$('#heist-body button');
    let clicked=false;
    for (const btn of btns){ const t=await btn.innerText(); if (t.includes('合作')){ await btn.click(); clicked=true; break; } }
    await p.waitForTimeout(3500);
    console.log('第'+i+'次出手後視圖:', await active(), '| 有按到鈕:', clicked);
    const body = await p.$eval('#heist-body', el=>el.innerText.replace(/\n+/g,' | ').slice(0,110)).catch(()=>'');
    console.log('   畫面:', body);
  }
  await b.close();
})();
