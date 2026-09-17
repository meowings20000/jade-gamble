const { chromium } = require('playwright');
(async () => {
  const B = 'http://127.0.0.1:3002';
  const b = await chromium.launch();
  const ctx = await b.newContext();
  const p = await ctx.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push(e.message));
  await ctx.request.post(B + '/api/auth/mock', { data: { username: 'bui' + Date.now() } });
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(1800);
  await p.click('nav button[data-view="bank"]');
  await p.waitForTimeout(2000);
  const inputs = await p.$$eval('#view-bank input', els => els.map(e => ({ id: e.id, ph: e.placeholder, type: e.type })));
  console.log('輸入框:', JSON.stringify(inputs));
  const amount = (await p.$('#bank-amount')) || (await p.$('input[placeholder*="5000"]'));
  const hours = (await p.$('#bank-hours')) || (await p.$('input[placeholder*="小時"]'));
  const reason = (await p.$('#bank-reason')) || (await p.$('input[placeholder*="娘"]'));
  if (amount) await amount.fill('5000');
  if (hours) await hours.fill('3');
  if (reason) await reason.fill('我要周轉買石頭');
  for (const e of await p.$$('#view-bank button')) {
    const t = await e.innerText();
    if (t.includes('提出申請')) { await e.click(); break; }
  }
  await p.waitForTimeout(9000);
  const offer = await p.$eval('#bank-offer', el => el.innerText.replace(/\n+/g, ' | ').trim()).catch(() => '(讀不到)');
  console.log('提議卡:', offer.slice(0, 240));
  const btns = await p.$$eval('#view-bank button', els => els.map(e => e.innerText.trim()));
  console.log('按鈕清單:', JSON.stringify(btns));
  let acc = null;
  for (const e of await p.$$('#view-bank button')) {
    const t = (await e.innerText()).trim();
    if (t === '接受' || t.indexOf('接受') === 0) { acc = e; break; }
  }
  if (acc) {
    await acc.click();
    await p.waitForTimeout(2500);
    console.log('按接受後喵喵幣:', await p.$eval('#chips', el => el.innerText.trim()).catch(() => '?'));
    console.log('目前貸款:', await p.$eval('#bank-current', el => el.innerText.replace(/\n+/g, ' | ').trim()).catch(() => '?'));
  } else {
    console.log('!! 找不到「接受」按鈕（這就是卡住的原因）');
  }
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 3) : '無');
  await b.close();
})();
