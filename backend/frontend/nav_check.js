// nav_check.js — why can't the player see the 轉賬 button?
// Dumps every nav button's text/box/visibility at desktop and mobile widths.
const { chromium } = require('playwright');

const URL = process.env.JADE_URL || 'https://jade.meowmeow12245ouo.dpdns.org';

(async () => {
  const browser = await chromium.launch();
  for (const vp of [{ width: 1440, height: 900, tag: 'desktop-1440' }, { width: 390, height: 844, tag: 'mobile-390' }]) {
    const page = await browser.newPage({ viewport: vp });
    await page.goto(URL, { waitUntil: 'networkidle' });
    const mock = page.locator('#login-mock');
    if (await mock.isVisible()) { await mock.click(); await page.waitForTimeout(800); }

    const info = await page.evaluate(() => {
      const nav = document.querySelector('#nav');
      const out = {
        innerWidth: innerWidth,
        navDisplay: getComputedStyle(nav).display,
        navRect: nav.getBoundingClientRect().toJSON(),
        navScroll: { sw: nav.scrollWidth, cw: nav.clientWidth, sl: nav.scrollLeft },
        bodyMobile: document.body.className,
        buttons: [],
      };
      for (const b of document.querySelectorAll('#nav button, header #balance button')) {
        const r = b.getBoundingClientRect();
        out.buttons.push({
          view: b.dataset.view || b.id,
          text: b.textContent.trim(),
          x: Math.round(r.x), w: Math.round(r.width),
          display: getComputedStyle(b).display,
          visibility: getComputedStyle(b).visibility,
          inViewport: r.x >= 0 && r.x + r.width <= innerWidth + 1,
        });
      }
      return out;
    });
    console.log(`\n=== ${vp.tag} ===`);
    console.log(JSON.stringify(info, null, 1));
    await page.screenshot({ path: `nav-${vp.tag}.png`, fullPage: false });
    await page.close();
  }
  await browser.close();
})();
