// mobile_check.js — programmatic mobile-layout verification against the live
// stack using Edge headless (this machine's only Chromium). Asserts:
//   1. no horizontal overflow at 390px on every view
//   2. all nav/mode buttons visible and inside the viewport
//   3. modals (classic bet) fit within the viewport height
//   4. touch targets >= 36px on primary action buttons
const { chromium } = require('playwright');
const assert = require('assert');

(async () => {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
  await page.goto('http://127.0.0.1:3002/', { waitUntil: 'networkidle' });

  // login as guest
  const mockBtn = page.locator('#login-mock');
  if (await mockBtn.isVisible()) {
    await mockBtn.click();
    await page.waitForTimeout(600);
  }

  const checkOverflow = async (label) => {
    const over = await page.evaluate(() => {
      const d = document.documentElement;
      return { sw: d.scrollWidth, cw: d.clientWidth };
    });
    assert.ok(over.sw <= over.cw + 1, `${label}: horizontal overflow ${over.sw} > ${over.cw}`);
    console.log(`✓ ${label}: no horizontal overflow (${over.sw} <= ${over.cw})`);
  };

  // every view
  const views = ['shop', 'warehouse', 'market', 'exchange', 'collection', 'ranks'];
  for (const v of views) {
    await page.click(`nav button[data-view="${v}"]`);
    await page.waitForTimeout(700);
    await checkOverflow(v);
  }

  // mode buttons visible & inside viewport when logged in
  for (const sel of ['#classic-bet', '#yboss-bet', '#lang-toggle']) {
    const el = page.locator(sel);
    const visible = await el.isVisible();
    assert.ok(visible, `${sel} not visible`);
    // nav strip scrolls horizontally on mobile: scroll the target into view
    await el.evaluate(n => n.scrollIntoView({ block: 'nearest', inline: 'center' }));
    await page.waitForTimeout(150);
    const box = await el.boundingBox();
    assert.ok(box && box.x >= -1 && box.x + box.width <= 391, `${sel} not scrollable into view: ${JSON.stringify(box)}`);
    console.log(`✓ ${sel} reachable (${Math.round(box.x)},${Math.round(box.y)},${Math.round(box.width)}w)`);
  }

  // touch target size: nav buttons and action buttons >= 36px tall
  const heights = await page.evaluate(() => {
    const els = [...document.querySelectorAll('nav button, .stone-card .btn')];
    return els.slice(0, 20).map(e => ({ t: (e.textContent || '').trim().slice(0, 6), h: e.getBoundingClientRect().height }));
  });
  const small = heights.filter(h => h.h < 36 && h.h > 0);
  assert.ok(small.length === 0, `touch targets too small: ${JSON.stringify(small)}`);
  console.log(`✓ touch targets >= 36px (${heights.length} buttons checked)`);

  // modal fits: open classic bet modal, measure
  await page.click('#classic-bet');
  await page.waitForTimeout(300);
  const modalBox = await page.locator('.modal').boundingBox();
  assert.ok(modalBox, 'classic modal not open');
  assert.ok(modalBox.y + modalBox.height <= 844 + 1, `modal exceeds viewport: y=${modalBox.y} h=${modalBox.height}`);
  console.log(`✓ classic modal fits viewport (y=${Math.round(modalBox.y)}, h=${Math.round(modalBox.height)})`);
  await page.keyboard.press('Escape');
  await page.locator('.modal-bg').click({ position: { x: 5, y: 5 } }).catch(() => {});

  // screenshot for the record
  await page.screenshot({ path: 'mobile-390.png', fullPage: false });
  console.log('✓ screenshot mobile-390.png saved');

  // desktop sanity: nothing broke at 1280px
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.waitForTimeout(300);
  await checkOverflow('desktop-1280');

  await browser.close();
  console.log('\nALL MOBILE CHECKS PASSED');
})().catch(e => { console.error('FAIL:', e.message); process.exit(1); });