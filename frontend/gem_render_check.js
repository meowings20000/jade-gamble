// gem_render_check.js — 把 6 種寶石用真實的 cutView 畫出來，檢查顏色與存圖。
const { chromium } = require('playwright');
const URL = process.env.JADE_URL || 'http://127.0.0.1:3002';

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1200, height: 520 } });
  page.on('pageerror', (e) => console.log('[pageerror]', e.message));
  await page.goto(URL, { waitUntil: 'networkidle' });

  const res = await page.evaluate(() => {
    const keys = ['amethyst', 'topaz', 'sapphire', 'ruby', 'emerald', 'diamond'];
    const out = [];
    document.body.innerHTML = '<div id="strip" style="display:flex;gap:8px;padding:10px;background:#141210"></div>';
    for (const k of keys) {
      const cv = document.createElement('canvas');
      cv.style.width = '180px'; cv.style.height = '220px';
      document.getElementById('strip').appendChild(cv);
      StoneRender.cutView(cv, '12345678901234', k, 'base', { grade: 0 });
      const ctx = cv.getContext('2d');
      const d = ctx.getImageData(cv.width * 0.55, cv.height * 0.3, cv.width * 0.35, cv.height * 0.4).data;
      let r = 0, g = 0, b = 0, n = 0, bright = 0;
      for (let i = 0; i < d.length; i += 4) {
        r += d[i]; g += d[i + 1]; b += d[i + 2]; n++;
        if (d[i] + d[i + 1] + d[i + 2] > 600) bright++;
      }
      out.push({ key: k, r: Math.round(r / n), g: Math.round(g / n), b: Math.round(b / n), brightPx: bright });
    }
    // 標籤
    const labels = ['紫水晶', '黃玉', '藍寶石', '紅寶石', '祖母綠', '鑽石'];
    document.getElementById('strip').querySelectorAll('canvas').forEach((cv, i) => {
      const d = document.createElement('div');
      d.textContent = labels[i];
      d.style.cssText = 'color:#d4a94e;font:13px sans-serif;text-align:center;width:180px';
      cv.insertAdjacentElement('afterend', d);
    });
    return out;
  });
  console.log('寶石切面平均色（R,G,B）與亮點數：');
  for (const s of res) console.log(`  ${s.key.padEnd(9)} R=${String(s.r).padStart(3)} G=${String(s.g).padStart(3)} B=${String(s.b).padStart(3)}  亮點=${s.brightPx}`);
  const dom = { ruby: 'r', emerald: 'g', sapphire: 'b' };
  let ok = true;
  for (const [k, ch] of Object.entries(dom)) {
    const s = res.find((x) => x.key === k);
    const vals = { r: s.r, g: s.g, b: s.b };
    const top = Object.keys(vals).sort((a, b) => vals[b] - vals[a])[0];
    const pass = top === ch;
    if (!pass) ok = false;
    console.log(`${pass ? 'PASS' : 'FAIL'}  ${k} 主色應該是 ${ch.toUpperCase()}（實際最高 ${top.toUpperCase()}）`);
  }
  const sparkly = res.every((s) => s.brightPx > 20);
  console.log(`${sparkly ? 'PASS' : 'FAIL'}  每顆都有閃光亮點（>20 像素）`);
  await page.screenshot({ path: 'gem-strip.png' });
  console.log('screenshot: gem-strip.png');
  await browser.close();
})();
