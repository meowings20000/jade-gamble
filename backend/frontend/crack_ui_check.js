// crack_ui_check.js — 用真實 UI 流程驗證：刮開的格子看得見裂紋嗎？
//   1. 登入 → 買一顆公斤料 → 進倉庫 → 開刮石面板
//   2. 用滑鼠逐格點開（走真正的 onCellReveal→markRevealed 路徑）
//   3. 讀真 canvas 的像素，比較「裂縫格」與「乾淨格」的暗度，並存截圖
const { chromium } = require('playwright');
const URL = process.env.JADE_URL || 'https://jade.meowmeow12245ouo.dpdns.org';

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1000, height: 950 } });
  page.on('pageerror', (e) => console.log('[pageerror]', e.message));
  const reveals = [];
  page.on('response', async (r) => {
    if (r.url().includes('/api/scratch/reveal')) {
      try { const j = await r.json(); reveals.push({ cell: j.cell, kind: j.kind }); } catch {}
    }
  });
  await page.goto(URL, { waitUntil: 'networkidle' });
  const mock = page.locator('#login-mock');
  if (await mock.isVisible()) { await mock.click(); await page.waitForTimeout(800); }

  const stone = await page.evaluate(async () => {
    const shop = await (await fetch('/api/shop')).json();
    const it = shop.grades[0].items[0];
    await fetch('/api/shop/buy', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ stone_id: it.id }),
    });
    return { id: it.id, seed: it.seed, grade: it.grade };
  });
  console.log('bought kilo stone', stone.id);

  // 倉庫 → 刮
  await page.click('nav button[data-view="warehouse"]');
  await page.waitForTimeout(800);
  const btn = page.locator('button:has-text("刮")').first();
  await btn.click();
  await page.waitForTimeout(900);
  const canvas = page.locator('#scr-cv');
  if (!(await canvas.count())) { console.log('沒開到刮石面板'); await browser.close(); return; }
  const box = await canvas.boundingBox();
  console.log('canvas box', JSON.stringify(box));

  // 逐格點開（4 欄 × 3 列）
  const kinds = [];
  for (let c = 0; c < 8; c++) {
    const col = c % 4, row = Math.floor(c / 4);
    const x = box.x + box.width * ((col + 0.5) / 4);
    const y = box.y + box.height * ((row + 0.5) / 3);
    const before = await page.evaluate(() => window.__kinds ? window.__kinds.length : 0);
    await page.mouse.click(x, y);
    await page.waitForTimeout(700);
    kinds.push(c);
    if (!(await page.locator('#scr-cv').count())) { console.log('面板提早關閉於 cell', c); break; }
  }
  await page.screenshot({ path: 'crack-ui-8cells.png' });
  // 12 格全開面板會自動關掉：先只刮 8 格，這時讀得到畫面
  await page.waitForTimeout(1200);

  const stats = await page.evaluate(() => {
    const cv = document.querySelector('#scr-cv');
    const ctx = cv.getContext('2d');
    const dpr = cv.width / cv.getBoundingClientRect().width;
    const cols = 4, rows = 3;
    const cw = cv.width / cols, ch = cv.height / rows;
    const out = [];
    for (let c = 0; c < 12; c++) {
      const x0 = (c % cols) * cw, y0 = Math.floor(c / cols) * ch;
      const d = ctx.getImageData(x0 + cw * 0.1, y0 + ch * 0.1, cw * 0.8, ch * 0.8).data;
      let sum = 0, n = 0, dark = 0, red = 0;
      for (let i = 0; i < d.length; i += 4) {
        const lum = 0.299 * d[i] + 0.587 * d[i + 1] + 0.114 * d[i + 2];
        sum += lum; n++;
        if (lum < 70) dark++;
        if (d[i] > d[i + 2] + 30) red++;
      }
      out.push({ cell: c, avgLum: +(sum / n).toFixed(1), darkPct: +(100 * dark / n).toFixed(2), redPct: +(100 * red / n).toFixed(2) });
    }
    return { out, dpr };
  });
  const crackCells = reveals.filter((r) => r.kind !== 'clean').map((r) => r.cell);
  console.log('revealed (server truth):', JSON.stringify(reveals));
  console.log('cell pixel stats:');
  for (const s of stats.out) {
    const tag = crackCells.includes(s.cell) ? '裂' : '  ';
    console.log(`  ${tag} cell ${s.cell}: avgLum=${s.avgLum} dark%=${s.darkPct} red%=${s.redPct}`);
  }
  if (crackCells.length) {
    const cr = stats.out.filter((s) => crackCells.includes(s.cell));
    const cl = stats.out.filter((s) => !crackCells.includes(s.cell) && s.avgLum > 0);
    const avg = (a, k) => a.length ? a.reduce((x, y) => x + y[k], 0) / a.length : 0;
    console.log(`裂縫格 dark% ${avg(cr, 'darkPct').toFixed(2)}（n=${cr.length}） vs 已刮乾淨格 ${avg(cl, 'darkPct').toFixed(2)}（n=${cl.length}）`);
    console.log(avg(cr, 'darkPct') > avg(cl, 'darkPct') + 2 ? '✓ 裂縫格明顯比乾淨格暗 → 刮開看得到裂紋' : '✗ 裂紋仍然看不出來');
  } else {
    console.log('這顆石頭刮開的格子裡沒有裂紋（換一顆再測）');
  }
  await page.screenshot({ path: 'crack-ui.png' });
  console.log('screenshot: crack-ui.png');
  await browser.close();
})();
