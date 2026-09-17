
const { chromium } = require('playwright');
(async () => {
  const B='http://127.0.0.1:3002';
  const b=await chromium.launch(); const ctx=await b.newContext(); const p=await ctx.newPage();
  const errs=[]; p.on('pageerror', e=>errs.push(e.message));
  p.on('console', m=>{ if (m.type()==='error') errs.push('console: '+m.text().slice(0,140)); });
  await ctx.request.post(B+'/api/auth/mock',{data:{username:'bkp'+Date.now()}});
  await p.goto(B+'/',{waitUntil:'domcontentloaded'}); await p.waitForTimeout(1800);
  await p.click('nav button[data-view="bank"]'); await p.waitForTimeout(2500);
  const txt = async (sel) => await p.$eval(sel, el => el.innerText.replace(/\n+/g,' | ').trim()).catch(() => '(元素不存在)');
  console.log('條款卡:', await txt('#bank-terms'));
  console.log('提議卡:', await txt('#bank-offer'), '| 元素存在:', !!(await p.$('#bank-offer')));
  console.log('目前貸款:', await txt('#bank-current'));
  console.log('按鈕:', JSON.stringify(await p.$$eval('#view-bank button', els => els.map(e=>e.innerText.trim()))));
  console.log('JS 錯誤:', errs.length ? errs.slice(0,4) : '無');
  await b.close();
})();
