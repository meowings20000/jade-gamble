// crack_visual_check.js — 刮開的格子裡到底看不看得見裂紋？
//
// 做法：真的用瀏覽器刮（透過程式化的前端 API），刮到伺服器回報 crack 的格子，
// 然後在該格與一個乾淨格各取樣像素，比較暗度／紅色分量。順便存截圖。
const { chromium } = require('playwright');

const URL = process.env.JADE_URL || 'https://jade.meowmeow12245ouo.dpdns.org';

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1100, height: 900 } });
  const errs = [];
  page.on('pageerror', (e) => errs.push(e.message));
  page.on('console', (m) => { if (m.type() === 'error') errs.push(m.text()); });
  await page.goto(URL, { waitUntil: 'networkidle' });
  const mock = page.locator('#login-mock');
  if (await mock.isVisible()) { await mock.click(); await page.waitForTimeout(700); }

  // 買一顆公斤料（最便宜），開刮石
  const bought = await page.evaluate(async () => {
    const shop = await (await fetch('/api/shop')).json();
    const item = shop.grades[0].items[0];
    const buy = await (await fetch('/api/shop/buy', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ stone_id: item.id }),
    })).json();
    return { stoneId: item.id, price: item.price, chips: buy.chips, hint: item.hint };
  });
  console.log('kilo stone:', JSON.stringify(bought));
  if (bought.hint) { console.log('!! 蒙頭料居然有免費描述:', bought.hint); }

  const start = await page.evaluate(async (sid) => (await fetch('/api/scratch/start', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ stone_id: sid }),
  })).json(), bought.stoneId);
  console.log('scratch start hint:', start.hint);

  // 逐格刮開，記下每格的 kind
  const kinds = [];
  for (let i = 0; i < 12; i++) {
    const r = await page.evaluate(async ([sid, cell]) => (await fetch('/api/scratch/reveal', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ stone_id: sid, cell }),
    })).json(), [bought.stoneId, i]);
    kinds.push({ cell: i, kind: r.kind });
  }
  console.log('revealed kinds:', JSON.stringify(kinds));

  const crackCells = kinds.filter((k) => k.kind !== 'clean').map((k) => k.cell);
  if (!crackCells.length) {
    // 這顆石頭沒裂紋：換一顆再試（最多 4 顆）
    console.log('這顆沒裂紋，換下一顆…');
  }

  // 用前端 renderer 重畫同一顆石頭的刮板，檢查裂紋格的像素
  const res = await page.evaluate(async ([sid, crackCells]) => {
    const start = await (await fetch('/api/scratch/start', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ stone_id: sid }),
    })).json();
    const canvas = document.createElement('canvas');
    canvas.width = 480; canvas.height = 360;
    canvas.style.position = 'fixed'; canvas.style.left = '0'; canvas.style.top = '0';
    canvas.style.zIndex = '9999';
    document.body.appendChild(canvas);
    const board = new StoneRender.ScratchBoard(canvas, { seed: start.seed, grade: start.grade, cells: 12 });
    for (const rv of (start.revealed || [])) board.markRevealed(rv.cell, rv.kind);
    board.render && board.render();
    // 刮開全部（跳過動畫）
    for (let i = 0; i < 12; i++) {
      const r = await (await fetch('/api/scratch/reveal', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ stone_id: sid, cell: i }),
      }));
      const j = await r.json();
      board.markRevealed(i, j.kind);
    }
    board.render && board.render();
    // 取樣：找出每個格子的平均亮度
    const ctx = canvas.getContext('2d');
    const cols = 4, rows = 3, cw = canvas.width / cols, ch = canvas.height / rows;
    const stats = [];
    for (let c = 0; c < 12; c++) {
      const x0 = (c % cols) * cw, y0 = Math.floor(c / cols) * ch;
      const d = ctx.getImageData(x0 + cw * 0.15, y0 + ch * 0.15, cw * 0.7, ch * 0.7).data;
      let sum = 0, n = 0, dark = 0, reddish = 0;
      for (let i = 0; i < d.length; i += 4) {
        const lum = 0.299 * d[i] + 0.587 * d[i + 1] + 0.114 * d[i + 2];
        sum += lum; n++;
        if (lum < 60) dark++;
        if (d[i] > d[i + 2] + 25) reddish++;
      }
      stats.push({ cell: c, avgLum: +(sum / n).toFixed(1), darkPct: +(100 * dark / n).toFixed(1), redPct: +(100 * reddish / n).toFixed(1) });
    }
    return { stats, crackCells };
  }, [bought.stoneId, crackCells]);

  console.log('cell stats:');
  for (const s of res.stats) {
    const isCrack = res.crackCells.includes(s.cell);
    console.log(`  cell ${s.cell} ${isCrack ? '裂' : '  '} avgLum=${s.avgLum} dark%=${s.darkPct} red%=${s.redPct}`);
  }
  if (res.crackCells.length) {
    const cr = res.stats.filter((s) => res.crackCells.includes(s.cell));
    const cl = res.stats.filter((s) => !res.crackCells.includes(s.cell));
    const avg = (arr, k) => arr.reduce((a, b) => a + b[k], 0) / arr.length;
    console.log(`裂縫格 dark% 平均 ${avg(cr, 'darkPct').toFixed(1)} vs 乾淨格 ${avg(cl, 'darkPct').toFixed(1)}`);
    console.log(avg(cr, 'darkPct') > avg(cl, 'darkPct') + 3 ? '✓ 裂縫格明顯比乾淨格暗（看得到裂紋）' : '✗ 裂縫看不出來');
  }
  await page.screenshot({ path: 'crack-check.png' });
  if (errs.length) console.log('page errors:', errs.slice(0, 5));
  await browser.close();
})();
