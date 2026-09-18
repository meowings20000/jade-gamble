
const { chromium } = require('playwright');
const B='http://127.0.0.1:3002';
(async()=>{
  const b=await chromium.launch(); const ct=await b.newContext(); const p=await ct.newPage();
  const errs=[]; p.on('pageerror',e=>errs.push('PAGEERROR: '+e.message));
  await ct.request.post(B+'/api/auth/mock',{data:{username:'cos'+Date.now()}});
  await p.goto(B+'/',{waitUntil:'domcontentloaded'}); await p.waitForTimeout(2200);
  const bg0 = await p.evaluate(()=>getComputedStyle(document.body).backgroundColor);
  // 買翠玉桌布
  let r = await ct.request.post(B+'/api/exchange/buy',{data:{key:'theme_jade'}});
  console.log('買 theme_jade:', r.status());
  await p.reload({waitUntil:'domcontentloaded'}); await p.waitForTimeout(2500);
  const bg1 = await p.evaluate(()=>getComputedStyle(document.body).backgroundColor);
  console.log('body class:', await p.evaluate(()=>document.body.className), '| 背景色', bg0, '→', bg1);
  // 買彩帶
  r = await ct.request.post(B+'/api/exchange/buy',{data:{key:'fx_confetti'}});
  console.log('買 fx_confetti:', r.status());
  await p.reload({waitUntil:'domcontentloaded'}); await p.waitForTimeout(2500);
  console.log('window.__hasFX:', await p.evaluate(()=>window.__hasFX));
  await p.evaluate(()=>confettiBurst());
  await p.waitForTimeout(400);
  console.log('彩帶元素數:', await p.$$eval('.confetti', e=>e.length));
  // 交換所清單
  await p.click('nav button[data-view="exchange"]',{force:true}); await p.waitForTimeout(1800);
  const items = await p.$$eval('#exchange-list .card h3', e=>e.map(x=>x.innerText.split('\n')[0].trim()));
  console.log('兌換所共', items.length, '件:', JSON.stringify(items.slice(-9)));
  console.log('JS 錯誤:', errs.slice(0,3));
  await b.close();
})();
