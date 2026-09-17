const { chromium } = require('playwright');
const B = 'http://127.0.0.1:3002';
(async () => {
  const b = await chromium.launch();
  const ct = await b.newContext({ viewport: { width: 1280, height: 900 } });
  const p = await ct.newPage();
  const errs = [];
  p.on('pageerror', e => errs.push('PAGEERROR: ' + e.message));
  await ct.request.post(B + '/api/auth/mock', { data: { username: 'fx' + Date.now() } });
  const buy = async () => {
    const shop = await (await ct.request.get(B + '/api/shop')).json();
    let best = null;
    for (const g of shop.grades || []) for (const it of g.items || []) if (!best || it.price < best.price) best = it;
    await ct.request.post(B + '/api/shop/buy', { data: { stone_id: best.id } });
    return best;
  };
  const clear = () => p.evaluate(() => { const m = document.querySelector('.modal-bg'); if (m) m.remove(); });
  const modal = () => p.$eval('.modal', el => el.innerText.replace(/\s+/g, ' ').slice(0, 70)).catch(() => '(沒面板)');
  const btns = () => p.$$eval('.modal button', e => e.map(x => x.innerText.trim()));

  const A = await buy();
  const Bst = await buy();
  await p.goto(B + '/', { waitUntil: 'domcontentloaded' });
  await p.waitForTimeout(2000);

  // 1) A：選輕磨（還沒磨過）→ 應該看到「換力度」
  await clear(); await p.evaluate((st) => doPolish(st), A); await p.waitForTimeout(1200);
  await p.evaluate(() => doPolishForcePicker && null);
  // 選力度面板 -> 按輕磨
  for (const e of await p.$$('.modal button')) { if ((await e.innerText()).includes('輕磨')) { await e.click({ force: true }); break; } }
  await p.waitForTimeout(2500);
  console.log('A 選輕磨後 →', await modal());
  console.log('   按鈕:', JSON.stringify(await btns()));
  // 按「換力度」
  let rf = null;
  for (const e of await p.$$('.modal button')) { if ((await e.innerText()).includes('換力度')) { rf = e; break; } }
  if (rf) { await rf.click({ force: true }); await p.waitForTimeout(1200); console.log('   按「換力度」→', await modal()); }

  // 2) 真的磨一層後，「換力度」應該消失
  for (const e of await p.$$('.modal button')) { if ((await e.innerText()).includes('輕磨')) { await e.click({ force: true }); break; } }
  await p.waitForTimeout(2000);
  let adv = null;
  for (const e of await p.$$('.modal button')) { if ((await e.innerText()).includes('再磨一層')) { adv = e; break; } }
  if (adv) { await adv.click({ force: true }); await p.waitForTimeout(2500); console.log('磨過一層後 →', await modal()); console.log('   按鈕:', JSON.stringify(await btns())); }
  const back = await ct.request.post(B + '/api/polish/start', { data: { stone_id: A.id, force: 3 } });
  console.log('磨過一層後想改力度:', back.status(), JSON.stringify(await back.json()).slice(0, 80));

  // 3) 手上還有 A 在磨時，去開 B 應該被擋
  await clear(); await p.evaluate((st) => doPolish(st), Bst); await p.waitForTimeout(1500);
  console.log('B（A 還在磨）→', await modal());
  console.log('JS 錯誤:', errs.length ? errs.slice(0, 4) : '無');
  await b.close();
})();
