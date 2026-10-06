
const { chromium } = require('playwright');
const B='http://127.0.0.1:3002';
(async()=>{
  const b=await chromium.launch(); const ct=await b.newContext(); const p=await ct.newPage();
  const errs=[]; p.on('pageerror',e=>errs.push('PAGEERROR: '+e.message));
  p.on('console',m=>{ if(m.type()==='error') errs.push('CONSOLE: '+m.text().slice(0,150)); });
  await ct.request.post(B+'/api/auth/mock',{data:{username:'dbg'+Date.now()}});
  const shop=await (await ct.request.get(B+'/api/shop')).json();
  let best=null; for(const g of shop.grades||[]) for(const it of g.items||[]) if(!best||it.price<best.price) best=it;
  await ct.request.post(B+'/api/shop/buy',{data:{stone_id:best.id}});
  await ct.request.post(B+'/api/polish/start',{data:{stone_id:best.id,force:2}});
  await p.goto(B+'/',{waitUntil:'domcontentloaded'}); await p.waitForTimeout(2000);
  console.log('頁面 /api/me:', JSON.stringify(await p.evaluate(async()=>{ const m=await api('GET','/api/me'); return {running:m.polish_running, stones:m.polish_stones, stone:m.polish_stone}; })));
  console.log('頁面直接結算:', await p.evaluate(async(id)=>{ try{ const r=await api('POST','/api/polish/cash',{stone_id:id}); return JSON.stringify(r).slice(0,140); }catch(e){ return 'ERR: '+(e&&e.message); } }, best.id));
  console.log('錯誤:', errs.slice(0,3));
  await b.close();
})();
