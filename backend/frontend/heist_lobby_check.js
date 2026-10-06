
const { chromium } = require('playwright');
(async () => {
  const B='http://127.0.0.1:3002';
  const b=await chromium.launch(); const ctx=await b.newContext();
  // A：開一桌等人（製造排隊中的桌）
  const A = await ctx.newPage();
  await ctx.request.post(B+'/api/auth/mock',{data:{username:'lobbyA'+Date.now()}});
  await A.goto(B+'/',{waitUntil:'domcontentloaded'}); await A.waitForTimeout(1500);
  await A.click('nav button[data-view="heist"]'); await A.waitForTimeout(1200);
  await (await A.$('#heist-body button')).click(); await A.waitForTimeout(1500);
  // B：換一組 cookie 看大廳（模擬另一個玩家）
  const ctx2 = await b.newContext();
  const Bp = await ctx2.newPage();
  await Bp.on('pageerror', e=>console.log('JS ERROR:', e.message));
  await ctx2.request.post(B+'/api/auth/mock',{data:{username:'lobbyB'+Date.now()}});
  await Bp.goto(B+'/',{waitUntil:'domcontentloaded'}); await Bp.waitForTimeout(1500);
  await Bp.click('nav button[data-view="heist"]'); await Bp.waitForTimeout(3000);
  const grab = async () => await Bp.$eval('#heist-body', el => (el.innerText.match(/排隊中[\s\S]*/)||[''])[0].replace(/\n+/g,' | ').slice(0,150)).catch(()=>'(讀不到)');
  const t1 = await grab();
  console.log('大廳排隊 t=0  :', t1);
  await Bp.waitForTimeout(6000);
  const t2 = await grab();
  console.log('大廳排隊 t=+6s:', t2);
  console.log('有即時更新:', t1 !== t2);
  await b.close();
})();
