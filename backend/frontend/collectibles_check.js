
const { chromium } = require('playwright');
const B='http://127.0.0.1:3002';
(async()=>{
  const b=await chromium.launch(); const ct=await b.newContext(); const p=await ct.newPage();
  const errs=[]; p.on('pageerror',e=>errs.push('PAGEERROR: '+e.message));
  await ct.request.post(B+'/api/auth/mock',{data:{username:'col'+Date.now()}});
  await p.goto(B+'/',{waitUntil:'domcontentloaded'}); await p.waitForTimeout(2500);
  // 兌換所
  await p.click('nav button[data-view="exchange"]',{force:true}); await p.waitForTimeout(2000);
  const items = await p.$$eval('#exchange-list .card h3', e=>e.map(x=>x.innerText.split('\n')[0].trim()));
  console.log('兌換所商品(%d):', items.length, JSON.stringify(items));
  // 稱號頁：顏色
  await p.click('nav button[data-view="ranks"]',{force:true}); await p.waitForTimeout(1200);
  await p.evaluate(()=>{ if (typeof loadTitles==='function') loadTitles(); });
  await p.waitForTimeout(2000);
  const titles = await p.$$eval('#title-grid .card div:first-child', e=>e.slice(0,6).map(x=>x.innerText.trim()+' / '+getComputedStyle(x).color));
  console.log('稱號前三筆顏色:', JSON.stringify(titles.slice(0,3)));
  // 裝一個稱號看看標題列顏色
  const eq = await ct.request.post(B+'/api/titles/equip',{data:{key:'newbie'}});
  await p.reload({waitUntil:'domcontentloaded'}); await p.waitForTimeout(2500);
  console.log('標題列:', (await p.$eval('#userbox', el=>el.innerText.trim())).slice(0,40));
  console.log('title_rare API:', JSON.stringify(await (await ct.request.get(B+'/api/me')).json()).match(/"title[^,]*/g));
  console.log('JS 錯誤:', errs.slice(0,3));
  await b.close();
})();
