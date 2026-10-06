const { chromium } = require('playwright');
(async () => {
  const B = 'http://127.0.0.1:3002';
  const b = await chromium.launch();
  const ctx = await b.newContext();
  const p = await ctx.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push(e.message));
  await ctx.request.post(B + '/api/auth/mock', { data: { username: 'gem' + Date.now() } });
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2500);

  try {
    await p.click('nav button[data-view="collection"]', { force: true, timeout: 8000 });
  } catch (e) { console.log('點圖鑑失敗:', e.message.split('\n')[0]); }
  await p.waitForTimeout(3000);
  const txt = await p.$eval('#gem-book', el => el.innerText.replace(/\n+/g, ' | ').trim()).catch(() => '(讀不到 #gem-book)');
  console.log('圖鑒:', txt.slice(0, 320));
  console.log('圖鑒卡片數:', await p.$$eval('#gem-book .card', els => els.length).catch(() => 0));

  await p.click('nav button[data-view="bank"]', { force: true, timeout: 8000 }).catch(() => {});
  await p.waitForTimeout(2000);
  await p.$eval('#bk-amount', el => { el.value = '5000'; }).catch(() => {});
  await p.$eval('#bk-hours', el => { el.value = '3'; }).catch(() => {});
  await p.$eval('#bk-reason', el => { el.value = '我要買石頭'; }).catch(() => {});
  for (const e of await p.$$('#view-bank button')) { if ((await e.innerText()).includes('提出申請')) { await e.click({ force: true }); break; } }
  await p.waitForTimeout(9000);
  await p.$eval('#bk-say', el => { el.value = '利率可以低一點嗎喵'; }).catch(() => {});
  for (const e of await p.$$('#view-bank button')) { if ((await e.innerText()).trim() === '送出') { await e.click({ force: true }); break; } }
  await p.waitForTimeout(10000);
  const chat = await p.$eval('#bk-chat', el => el.innerText.replace(/\n+/g, ' | ').trim()).catch(() => '(讀不到 #bk-chat)');
  console.log('對話紀錄:', chat.slice(0, 340));
  console.log('對話氣泡數:', await p.$$eval('#bk-chat .chat-row', els => els.length).catch(() => 0));
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 3) : '無');
  await b.close();
})();
