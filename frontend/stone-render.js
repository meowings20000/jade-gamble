// stone-render.js — seed-driven procedural stone renderer (Canvas 2D).
// The server only ships {seed, grade, hint}; the SAME seed always paints
// the SAME stone on every client (market listings look identical).
(function (global) {
  'use strict';

  // mulberry64: deterministic PRNG. Seeds arrive as exact decimal strings
  // (uint64 in JS numbers collapses at 2^53 — that bug made every stone
  // look identical); we hash the string into a 32-bit seed.
  function hashSeed(str) {
    // FNV-1a 32-bit over the decimal string
    let h = 0x811c9dc5;
    for (let i = 0; i < str.length; i++) {
      h ^= str.charCodeAt(i);
      h = Math.imul(h, 0x01000193);
    }
    return h >>> 0;
  }

  function rngFrom(seed) {
    const s = (typeof seed === 'string') ? hashSeed(seed)
      : (seed | 0); // legacy numeric seeds (tests)
    let h = Math.imul(s ^ 0x9e3779b1, 0x85ebca6b) ^ 0xc2b2ae35;
    return function () {
      h = Math.imul(h ^ (h >>> 16), 0x21f0aaad);
      h = Math.imul(h ^ (h >>> 15), 0x735a2d97);
      h ^= h >>> 15;
      return (h >>> 0) / 4294967296;
    };
  }

  // Perceptual variety palettes: [base, mid, glow]
  const VARIETY_PALETTE = {
    base:        ['#7d8a7a', '#5c6b5a', '#a8b5a0'],
    violet:      ['#9b7bb8', '#6d4f8a', '#c9a8e8'],
    bluewater:   ['#5f8aa8', '#3c6480', '#9fc4dd'],
    whitegreen:  ['#b8c9b0', '#8aa07f', '#e8f0e0'],
    floatingblue: ['#6f9bb5', '#4a7390', '#b5d3e8'],
    inkgreen:    ['#22302a', '#101a15', '#3d5c48'],
    yellowgreen: ['#b5a848', '#8a7d2e', '#e8dc7a'],
    springpurple:['#a878b0', '#7a4f84', '#d8b5e0'],
    fortune3:    ['#b08858', '#8a6840', '#e8c898'],
    imperial:    ['#1f8a4c', '#0e5c30', '#5fe898'],
  };

  const GRADE_SKIN = {
    0: { base: '#8a7d6b', speck: '#6b5f4f', edge: '#57503f' },   // kilo: rough grey-brown
    1: { base: '#9a8a72', speck: '#7a6a52', edge: '#5f5540' },   // feature: warmer
    2: { base: '#a89a80', speck: '#8a7a5f', edge: '#6b5f4a' },   // window: lighter
  };

  // stoneBlob builds the irregular outline (polar noise polygon).
  function stoneBlob(r, cx, cy, rad) {
    const pts = [];
    const n = 14;
    const w1 = 0.75 + r() * 0.4, w2 = 0.6 + r() * 0.5, ph1 = r() * Math.PI * 2, ph2 = r() * Math.PI * 2;
    for (let i = 0; i < n; i++) {
      const a = (i / n) * Math.PI * 2;
      const wob = 1 + 0.08 * Math.sin(a * 3 + ph1) + 0.05 * Math.sin(a * 5 + ph2);
      pts.push([cx + Math.cos(a) * rad * wob, cy + Math.sin(a) * rad * wob * w1]);
    }
    return pts;
  }

  function pathPts(ctx, pts) {
    ctx.beginPath();
    ctx.moveTo(pts[0][0], pts[0][1]);
    for (let i = 1; i < pts.length; i++) {
      const p0 = pts[i === 0 ? pts.length - 1 : i - 1];
      const p1 = pts[i];
      const mx = (p0[0] + p1[0]) / 2, my = (p0[1] + p1[1]) / 2;
      ctx.quadraticCurveTo(p0[0], p0[1], mx, my);
    }
    ctx.closePath();
  }

  // drawSkin paints skin + speckle + (feature stones) pine-flower / python band.
  function drawSkin(ctx, seed, grade, w, h, revealedCells, cracks) {
    const r = rngFrom(seed);
    const cx = w / 2, cy = h / 2, rad = Math.min(w, h) * 0.42;
    const pts = stoneBlob(r, cx, cy, rad);

    // skin base with soft radial shading
    const skin = GRADE_SKIN[grade] || GRADE_SKIN[0];
    ctx.save();
    pathPts(ctx, pts);
    const g = ctx.createRadialGradient(cx - rad * 0.3, cy - rad * 0.3, rad * 0.2, cx, cy, rad * 1.1);
    g.addColorStop(0, skin.speck);
    g.addColorStop(1, skin.edge);
    ctx.fillStyle = g;
    ctx.fill();
    ctx.clip();

    // speckle noise
    for (let i = 0; i < 260; i++) {
      const x = r() * w, y = r() * h;
      ctx.fillStyle = i % 3 ? skin.base : skin.speck;
      ctx.globalAlpha = 0.13 + r() * 0.2;
      const s = 0.5 + r() * 2.2;
      ctx.fillRect(x, y, s, s);
    }
    ctx.globalAlpha = 1;

    // feature-grade skin signals (honest visual clues)
    if (grade >= 1) {
      if (r() < 0.6) { // 松花 pine-flower: green moss patches
        const n = 2 + Math.floor(r() * 3);
        for (let i = 0; i < n; i++) {
          const x = cx + (r() - 0.5) * rad * 1.4, y = cy + (r() - 0.5) * rad * 1.4;
          const rr = 4 + r() * 10;
          const gg = ctx.createRadialGradient(x, y, 0, x, y, rr);
          gg.addColorStop(0, 'rgba(70,130,80,0.5)');
          gg.addColorStop(1, 'rgba(70,130,80,0)');
          ctx.fillStyle = gg;
          ctx.beginPath(); ctx.arc(x, y, rr, 0, Math.PI * 2); ctx.fill();
        }
      }
      if (r() < 0.5) { // 蟒带 python band: dark raised band
        ctx.strokeStyle = 'rgba(40,32,20,0.45)';
        ctx.lineWidth = 6 + r() * 8;
        const y0 = cy + (r() - 0.5) * rad;
        ctx.beginPath();
        ctx.moveTo(cx - rad, y0);
        ctx.bezierCurveTo(cx - rad * 0.4, y0 - 14, cx + rad * 0.4, y0 + 14, cx + rad, y0);
        ctx.stroke();
      }
    }

    // window cut (grade 2): a polished oval revealing interior
    if (grade === 2) {
      const wx = cx + (r() - 0.5) * rad * 0.5, wy = cy + (r() - 0.5) * rad * 0.5;
      const wr = rad * 0.32;
      const wg = ctx.createRadialGradient(wx, wy, 2, wx, wy, wr);
      wg.addColorStop(0, 'rgba(200,220,205,0.95)');
      wg.addColorStop(1, 'rgba(150,170,150,0.7)');
      ctx.fillStyle = wg;
      ctx.beginPath();
      ctx.ellipse(wx, wy, wr, wr * 0.75, r() * Math.PI, 0, Math.PI * 2);
      ctx.fill();
    }
    ctx.restore();
    return pts;
  }

  // drawInterior paints the revealed jade under the skin.
  // quality: brick|bean|oilgreen|icy|glass ; variety key as in VARIETY_PALETTE.
  function drawInterior(ctx, seed, w, h, opts) {
    const r = rngFrom(seed ^ 0x7f3a1c);
    const cx = w / 2, cy = h / 2, rad = Math.min(w, h) * 0.42;
    const pts = stoneBlob(rngFrom(seed), cx, cy, rad);
    const pal = VARIETY_PALETTE[opts.variety || 'base'] || VARIETY_PALETTE.base;

    // translucency by quality
    const alpha = { brick: 0.95, bean: 0.85, oilgreen: 0.7, icy: 0.5, glass: 0.35 }[opts.quality] || 0.8;

    ctx.save();
    pathPts(ctx, pts);
    ctx.clip();
    const g = ctx.createRadialGradient(cx - rad * 0.25, cy - rad * 0.25, rad * 0.1, cx, cy, rad);
    g.addColorStop(0, pal[2]);
    g.addColorStop(0.55, pal[0]);
    g.addColorStop(1, pal[1]);
    ctx.globalAlpha = alpha;
    ctx.fillStyle = g;
    ctx.fillRect(0, 0, w, h);
    ctx.globalAlpha = 1;

    // texture strands
    for (let i = 0; i < 40; i++) {
      ctx.strokeStyle = i % 2 ? pal[2] : pal[1];
      ctx.globalAlpha = 0.05 + r() * 0.1;
      ctx.lineWidth = 0.5 + r() * 1.4;
      const x = r() * w, y = r() * h;
      ctx.beginPath();
      ctx.moveTo(x, y);
      ctx.quadraticCurveTo(x + (r() - 0.5) * 40, y + (r() - 0.5) * 40, x + (r() - 0.5) * 70, y + (r() - 0.5) * 70);
      ctx.stroke();
    }
    ctx.globalAlpha = 1;

    // icy/glass: glass sheen streak
    if (opts.quality === 'icy' || opts.quality === 'glass') {
      const sg = ctx.createLinearGradient(cx - rad, cy - rad, cx + rad, cy + rad);
      sg.addColorStop(0, 'rgba(255,255,255,0)');
      sg.addColorStop(0.5, 'rgba(255,255,255,0.35)');
      sg.addColorStop(1, 'rgba(255,255,255,0)');
      ctx.fillStyle = sg;
      ctx.fillRect(0, 0, w, h);
    }
    ctx.restore();
    return pts;
  }

  // drawCracks overlays dark fracture polylines at given cell coords (12×8 grid).
  function drawCracks(ctx, seed, w, h, cells, deep) {
    const r = rngFrom(seed ^ 0xC4A0);
    const gw = w / 12, gh = h / 8;
    ctx.save();
    ctx.lineCap = 'round';
    for (const c of cells || []) {
      const x = (c % 12) * gw + gw / 2, y = Math.floor(c / 12) * gh + gh / 2;
      const isDeep = deep && deep.indexOf(c) >= 0;
      ctx.strokeStyle = 'rgba(20,16,12,0.85)';
      ctx.lineWidth = isDeep ? 3 : 1.6;
      ctx.beginPath();
      ctx.moveTo(x - gw, y - gh * 0.4);
      let px = x - gw, py = y - gh * 0.4;
      for (let s = 0; s < 3; s++) {
        const nx = px + gw * (0.5 + r() * 0.7), ny = py + (r() - 0.5) * gh * 1.2;
        ctx.lineTo(nx, ny);
        px = nx; py = ny;
      }
      ctx.stroke();
    }
    ctx.restore();
  }

  // drawLightCone: flashlight overlay (hint-driven colour, not truth).
  function drawLightCone(ctx, w, h, hue) {
    const cx = w / 2, cy = h / 2;
    const g = ctx.createRadialGradient(cx, cy, 4, cx, cy, Math.min(w, h) * 0.5);
    g.addColorStop(0, hue || 'rgba(255,244,200,0.55)');
    g.addColorStop(0.6, 'rgba(255,244,200,0.18)');
    g.addColorStop(1, 'rgba(255,244,200,0)');
    ctx.fillStyle = g;
    ctx.fillRect(0, 0, w, h);
  }

  // thumbnail: small canvas for shelf/inventory/market cards.
  function thumbnail(canvas, seed, grade) {
    const dpr = window.devicePixelRatio || 1;
    const w = canvas.width / dpr, h = canvas.height / dpr;
    canvas.width = w * dpr; canvas.height = h * dpr;
    const ctx = canvas.getContext('2d');
    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, w, h);
    drawSkin(ctx, seed, grade, w, h);
  }

  // revealView: big stone with scratch mask (revealed cells show interior).
  function revealView(canvas, seed, grade, revealed, opts) {
    const dpr = window.devicePixelRatio || 1;
    const w = canvas.clientWidth || 300, h = canvas.clientHeight || 200;
    canvas.width = w * dpr; canvas.height = h * dpr;
    const ctx = canvas.getContext('2d');
    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, w, h);
    // interior first
    drawInterior(ctx, seed, w, h, opts || {});
    // skin mask: paint skin, then punch out revealed cells
    // (offscreen mask approach)
    const mask = document.createElement('canvas');
    mask.width = w * dpr; mask.height = h * dpr;
    const mctx = mask.getContext('2d');
    mctx.scale(dpr, dpr);
    const pts = drawSkin(mctx, seed, grade, w, h);
    // destination-out the revealed cells
    mctx.globalCompositeOperation = 'destination-out';
    const gw = w / 4, gh = h / 4; // 4×3 = 12 cells for scratch
    for (const c of revealed || []) {
      const x = (c % 4) * gw, y = Math.floor(c / 4) * gh;
      mctx.beginPath();
      mctx.roundRect ? mctx.roundRect(x + 1, y + 1, gw - 2, gh - 2, 8) : mctx.rect(x + 1, y + 1, gw - 2, gh - 2);
      mctx.fill();
    }
    mctx.globalCompositeOperation = 'source-over';
    ctx.drawImage(mask, 0, 0, w, h);
    // cracks drawn only if opts.showCracks (after full reveal)
    if (opts && opts.showCracks) drawCracks(ctx, seed, w, h, opts.crackCells || [], opts.deepCells || []);
    return pts;
  }

  // ---------- ScratchBoard: fine-grained brush scratch UX ----------
  // Server model is unchanged (12 cells): the brush erases the skin
  // organically; a cell is *opened* (server call) once ~50% of its
  // in-blob area is brushed. Truth (cracks) is stamped only after the
  // server confirms the reveal — brushing can never leak crack cells.
  class ScratchBoard {
    constructor(canvas, seed, grade, opts) {
      this.canvas = canvas;
      this.seed = seed;
      this.grade = grade;
      this.opts = opts || {};
      this.cols = 4; this.rows = 3; this.cells = 12;
      this.revealed = new Set();
      this.pending = new Set();
      this.skinAlpha = 1;
      this.down = false;
      this.last = null;
      this.destroyed = false;

      const rect = canvas.getBoundingClientRect();
      this.w = rect.width || 480;
      this.h = rect.height || this.w * 0.75;
      this.dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.round(this.w * this.dpr);
      canvas.height = Math.round(this.h * this.dpr);
      this.ctx = canvas.getContext('2d');
      this.ctx.setTransform(this.dpr, 0, 0, this.dpr, 0, 0);
      canvas.style.touchAction = 'none';

      this.cw = this.w / this.cols;
      this.ch = this.h / this.rows;
      this.brushR = Math.max(9, Math.min(this.w, this.h) / 16);

      // blob identical to drawSkin's (same rng consumption order)
      const rb = rngFrom(this.seed);
      this.blob = stoneBlob(rb, this.w / 2, this.h / 2, Math.min(this.w, this.h) * 0.42);

      // coverage samples: 8×6 per cell; only in-blob samples count
      this.samples = [];
      for (let c = 0; c < this.cells; c++) {
        const arr = [];
        const x0 = (c % this.cols) * this.cw, y0 = Math.floor(c / this.cols) * this.ch;
        for (let sy = 0; sy < 6; sy++) for (let sx = 0; sx < 8; sx++) {
          const x = x0 + this.cw * (sx + 0.5) / 8;
          const y = y0 + this.ch * (sy + 0.5) / 6;
          arr.push({ x, y, in: this._inside(x, y), er: false });
        }
        this.samples.push(arr);
      }

      this._buildLayers();

      this._onDown = (e) => {
        e.preventDefault();
        this.down = true;
        this.last = null;
        const p = this._pt(e);
        this.downPos = { x: p.x, y: p.y, t: Date.now() };
        try { this.canvas.setPointerCapture(e.pointerId); } catch (err) { /* ok */ }
      };
      this._onMove = (e) => {
        if (!this.down) return;
        const p = this._pt(e);
        this.strokeTo(p.x, p.y);
        if (this.opts.onBrush) this.opts.onBrush();
      };
      this._onUp = (e) => {
        if (!this.down) return;
        this.down = false;
        this.last = null;
        const p = this._pt(e);
        const moved = Math.hypot(p.x - this.downPos.x, p.y - this.downPos.y);
        if (moved < 6 && Date.now() - this.downPos.t < 400) {
          this.autoScratch(this.cellAt(p.x, p.y));
        }
      };
      canvas.addEventListener('pointerdown', this._onDown);
      canvas.addEventListener('pointermove', this._onMove);
      canvas.addEventListener('pointerup', this._onUp);
      canvas.addEventListener('pointercancel', this._onUp);
      this._compose();
    }

    _pt(e) {
      const r = this.canvas.getBoundingClientRect();
      return {
        x: (e.clientX - r.left) * (this.w / r.width),
        y: (e.clientY - r.top) * (this.h / r.height),
      };
    }
    cellAt(x, y) {
      const cx = Math.min(this.cols - 1, Math.max(0, Math.floor(x / this.cw)));
      const cy = Math.min(this.rows - 1, Math.max(0, Math.floor(y / this.ch)));
      return cy * this.cols + cx;
    }
    _inside(x, y) {
      const pts = this.blob; let inside = false;
      for (let i = 0, j = pts.length - 1; i < pts.length; j = i++) {
        const xi = pts[i][0], yi = pts[i][1], xj = pts[j][0], yj = pts[j][1];
        if (((yi > y) !== (yj > y)) && (x < (xj - xi) * (y - yi) / (yj - yi) + xi)) inside = !inside;
      }
      return inside;
    }
    _layer() {
      const c = document.createElement('canvas');
      c.width = Math.round(this.w * this.dpr);
      c.height = Math.round(this.h * this.dpr);
      const ctx = c.getContext('2d');
      ctx.setTransform(this.dpr, 0, 0, this.dpr, 0, 0);
      return { el: c, ctx };
    }
    _buildLayers() {
      this.interior = this._layer();
      this.truth = this._layer();
      this.skin = this._layer();
      // generic jade under the skin — no crack info until server reveals
      drawInterior(this.interior.ctx, this.seed, this.w, this.h, {});
      drawSkin(this.skin.ctx, this.seed, this.grade, this.w, this.h);
    }
    _compose() {
      const ctx = this.ctx;
      ctx.clearRect(0, 0, this.w, this.h);
      ctx.drawImage(this.interior.el, 0, 0, this.w, this.h);
      // 石皮在真值層「下面」：刮開的裂紋／亮面才不會被皮蓋掉
      // （原本順序相反，所以刮開也看不到裂紋）。
      ctx.globalAlpha = this.skinAlpha;
      ctx.drawImage(this.skin.el, 0, 0, this.w, this.h);
      ctx.globalAlpha = 1;
      ctx.drawImage(this.truth.el, 0, 0, this.w, this.h);
    }
    brush(x, y) {
      const sc = this.skin.ctx;
      sc.save();
      sc.globalCompositeOperation = 'destination-out';
      const g = sc.createRadialGradient(x, y, this.brushR * 0.25, x, y, this.brushR);
      g.addColorStop(0, 'rgba(0,0,0,0.95)');
      g.addColorStop(0.75, 'rgba(0,0,0,0.5)');
      g.addColorStop(1, 'rgba(0,0,0,0)');
      sc.fillStyle = g;
      sc.beginPath();
      sc.arc(x, y, this.brushR, 0, Math.PI * 2);
      sc.fill();
      sc.restore();
      this._stampCoverage(x, y);
      this._compose();
    }
    strokeTo(x, y) {
      if (this.last) {
        const dx = x - this.last.x, dy = y - this.last.y;
        const d = Math.hypot(dx, dy);
        const step = Math.max(2, this.brushR * 0.4);
        const n = Math.max(1, Math.ceil(d / step));
        for (let i = 1; i <= n; i++) {
          this.brush(this.last.x + dx * i / n, this.last.y + dy * i / n);
        }
      } else {
        this.brush(x, y);
      }
      this.last = { x, y };
    }
    _stampCoverage(x, y) {
      const r = this.brushR, r2 = r * r;
      const c0 = Math.max(0, Math.floor((x - r) / this.cw));
      const c1 = Math.min(this.cols - 1, Math.floor((x + r) / this.cw));
      const r0 = Math.max(0, Math.floor((y - r) / this.ch));
      const r1 = Math.min(this.rows - 1, Math.floor((y + r) / this.ch));
      for (let cy = r0; cy <= r1; cy++) {
        for (let cx = c0; cx <= c1; cx++) {
          const c = cy * this.cols + cx;
          if (this.revealed.has(c) || this.pending.has(c)) continue;
          for (const s of this.samples[c]) {
            if (!s.in || s.er) continue;
            const dx = s.x - x, dy = s.y - y;
            if (dx * dx + dy * dy <= r2) s.er = true;
          }
          let tot = 0, er = 0;
          for (const s of this.samples[c]) {
            if (s.in) { tot++; if (s.er) er++; }
          }
          if (tot > 0 && er / tot >= 0.5) {
            this.pending.add(c);
            if (this.opts.onCellReveal) this.opts.onCellReveal(c);
          }
        }
      }
    }
    autoScratch(c) {
      if (this.destroyed || this.revealed.has(c) || this.pending.has(c)) return;
      const x0 = (c % this.cols) * this.cw, y0 = Math.floor(c / this.cols) * this.ch;
      let i = 0;
      const tick = () => {
        if (this.destroyed) return;
        const x = x0 + this.cw * (0.15 + 0.7 * Math.random());
        const y = y0 + this.ch * (0.15 + 0.7 * Math.random());
        this.brush(x, y);
        i++;
        if (i < 10) {
          setTimeout(tick, 26);
        } else if (!this.revealed.has(c) && !this.pending.has(c)) {
          // sparse corner-cell fallback: force the reveal
          this.pending.add(c);
          if (this.opts.onCellReveal) this.opts.onCellReveal(c);
        }
      };
      tick();
    }
    // markRevealed: server confirmed — stamp truth into the cell.
    markRevealed(cell, kind) {
      this.pending.delete(cell);
      this.revealed.add(cell);
      this._eraseCell(cell);
      this._stampTruth(cell, kind || 'clean');
      this._compose();
    }
    _eraseCell(c) {
      const sc = this.skin.ctx;
      const x0 = (c % this.cols) * this.cw, y0 = Math.floor(c / this.cols) * this.ch;
      sc.save();
      sc.globalCompositeOperation = 'destination-out';
      sc.fillStyle = 'rgba(0,0,0,1)';
      for (let i = 0; i < 14; i++) {
        const x = x0 + this.cw * (0.1 + 0.8 * Math.random());
        const y = y0 + this.ch * (0.1 + 0.8 * Math.random());
        sc.beginPath();
        sc.arc(x, y, this.cw * 0.22, 0, Math.PI * 2);
        sc.fill();
      }
      sc.restore();
    }
    _stampTruth(c, kind) {
      const t = this.truth.ctx;
      const x0 = (c % this.cols) * this.cw, y0 = Math.floor(c / this.cols) * this.ch;
      t.save();
      t.beginPath();
      t.rect(x0, y0, this.cw, this.ch);
      t.clip();
      if (kind === 'crack' || kind === 'deep_crack') {
        const deep = kind === 'deep_crack';
        const rr = rngFrom(this.seed ^ (0xC4A0 + c * 977));
        const pts = [];
        let px = x0 + this.cw * 0.04, py = y0 + this.ch * (0.25 + rr() * 0.5);
        pts.push([px, py]);
        for (let s = 0; s < 5; s++) {
          px += this.cw * (0.12 + rr() * 0.14);
          py = y0 + this.ch * (0.15 + rr() * 0.7);
          pts.push([px, py]);
        }
        const path = () => {
          t.beginPath();
          t.moveTo(pts[0][0], pts[0][1]);
          for (const p of pts.slice(1)) t.lineTo(p[0], p[1]);
        };
        t.lineCap = 'round';
        t.lineJoin = 'round';
        // 外圈：淡色「錯位」邊，讓裂縫在深色肉裡也看得出來
        t.strokeStyle = deep ? 'rgba(255,210,190,0.5)' : 'rgba(240,235,225,0.35)';
        t.lineWidth = deep ? 7 : 4.6;
        path();
        t.stroke();
        // 主線
        t.strokeStyle = deep ? 'rgba(72,8,6,0.98)' : 'rgba(28,20,14,0.95)';
        t.lineWidth = deep ? 4 : 2.4;
        path();
        t.stroke();
        if (deep) {
          const g = t.createRadialGradient(x0 + this.cw / 2, y0 + this.ch / 2, 2,
            x0 + this.cw / 2, y0 + this.ch / 2, this.cw * 0.6);
          g.addColorStop(0, 'rgba(150,30,20,0.28)');
          g.addColorStop(1, 'rgba(150,30,20,0)');
          t.fillStyle = g;
          t.fillRect(x0, y0, this.cw, this.ch);
        }
      } else {
        const g = t.createRadialGradient(x0 + this.cw / 2, y0 + this.ch / 2, 1,
          x0 + this.cw / 2, y0 + this.ch / 2, this.cw * 0.55);
        g.addColorStop(0, 'rgba(255,255,255,0.22)');
        g.addColorStop(1, 'rgba(255,255,255,0)');
        t.fillStyle = g;
        t.fillRect(x0, y0, this.cw, this.ch);
      }
      t.restore();
    }
    // finale: repaint with the TRUE quality/variety, then fade the skin.
    finale(quality, variety) {
      drawInterior(this.interior.ctx, this.seed, this.w, this.h, { quality, variety });
      const t0 = performance.now();
      const step = () => {
        if (this.destroyed) return;
        const k = Math.min(1, (performance.now() - t0) / 650);
        this.skinAlpha = 1 - k;
        this._compose();
        if (k < 1) requestAnimationFrame(step);
      };
      step();
    }
    destroy() {
      this.destroyed = true;
      this.canvas.removeEventListener('pointerdown', this._onDown);
      this.canvas.removeEventListener('pointermove', this._onMove);
      this.canvas.removeEventListener('pointerup', this._onUp);
      this.canvas.removeEventListener('pointercancel', this._onUp);
    }
  }



  // ---- 隱藏彩蛋：切出非玉石的寶石（純程式繪製，無素材）----
  const GEM_STYLE = {
    amethyst: { core: '#e0b3ff', deep: '#4b1d84', spark: 6, name: '紫水晶' },
    topaz: { core: '#ffeaa6', deep: '#8a6a12', spark: 6, name: '黃玉' },
    sapphire: { core: '#a8d8ff', deep: '#123f8a', spark: 8, name: '藍寶石' },
    ruby: { core: '#ff9d94', deep: '#8a0b1c', spark: 10, name: '紅寶石' },
    emerald: { core: '#a9ffd2', deep: '#0a5a34', spark: 12, name: '祖母綠' },
    diamond: { core: '#ffffff', deep: '#7fb6c8', spark: 16, name: '鑽石' },
  };

  function star(ctx, x, y, r, color) {
    ctx.save();
    ctx.strokeStyle = color;
    ctx.lineWidth = 1.2;
    ctx.beginPath();
    ctx.moveTo(x - r, y); ctx.lineTo(x + r, y);
    ctx.moveTo(x, y - r); ctx.lineTo(x, y + r);
    ctx.stroke();
    const g = ctx.createRadialGradient(x, y, 0, x, y, r);
    g.addColorStop(0, 'rgba(255,255,255,0.9)');
    g.addColorStop(1, 'rgba(255,255,255,0)');
    ctx.fillStyle = g;
    ctx.beginPath(); ctx.arc(x, y, r, 0, Math.PI * 2); ctx.fill();
    ctx.restore();
  }

  // gemCutView: 石皮外殼 + 右半切面露出一顆有切面與閃光的寶石。
  function gemCutView(ctx, seed, w, h, key, grade) {
    const st = GEM_STYLE[key];
    const rnd = rngFrom(seed + key);
    const cx = w * 0.5, cy = h * 0.5, rad = Math.min(w, h) * 0.40;
    const pts = stoneBlob(rngFrom(seed), cx, cy, rad);

    // 外殼：整顆石皮
    ctx.save(); pathPts(ctx, pts); ctx.clip();
    drawSkin(ctx, seed, grade != null ? grade : 0, w, h);
    ctx.restore();

    // 切面：右側露出一顆多面寶石
    ctx.save(); pathPts(ctx, pts); ctx.clip();
    const fx = cx + rad * 0.18, fy = cy, fr = rad * 0.78;
    const grad = ctx.createRadialGradient(fx - fr * 0.25, fy - fr * 0.3, fr * 0.04, fx, fy, fr);
    grad.addColorStop(0, st.core);
    grad.addColorStop(0.5, st.deep);
    grad.addColorStop(1, 'rgba(8,6,5,0.95)');
    const n = 7 + Math.floor(rnd() * 3);
    const verts = [];
    for (let i = 0; i < n; i++) {
      const a = (i / n) * Math.PI * 2 + rnd() * 0.25;
      const rr = fr * (0.72 + rnd() * 0.4);
      verts.push([fx + Math.cos(a) * rr, fy + Math.sin(a) * rr * 0.88]);
    }
    ctx.beginPath();
    verts.forEach(([x, y], i) => (i ? ctx.lineTo(x, y) : ctx.moveTo(x, y)));
    ctx.closePath();
    ctx.fillStyle = grad;
    ctx.fill();
    ctx.strokeStyle = 'rgba(255,255,255,0.4)';
    ctx.lineWidth = 1.4;
    ctx.stroke();
    // 切面線
    ctx.strokeStyle = 'rgba(255,255,255,0.22)';
    ctx.lineWidth = 1;
    for (const [x, y] of verts) {
      ctx.beginPath(); ctx.moveTo(fx, fy); ctx.lineTo(x, y); ctx.stroke();
    }
    ctx.restore();

    // 切線（刀刃進去的方向）
    ctx.strokeStyle = 'rgba(255,255,255,0.7)';
    ctx.lineWidth = 1.6;
    ctx.beginPath();
    ctx.moveTo(cx - rad * 0.15, cy - rad * 1.05);
    ctx.lineTo(cx - rad * 0.15, cy + rad * 1.05);
    ctx.stroke();

    // 閃光：越稀有越多
    for (let i = 0; i < st.spark; i++) {
      const a = rnd() * Math.PI * 2;
      const rr = rad * (0.2 + rnd() * 0.75);
      star(ctx, cx + Math.cos(a) * rr, cy + Math.sin(a) * rr * 0.8, 3 + rnd() * 7, st.core);
    }
  }

  // cutView: big cross-section — left half skin, right half revealed interior,
  // glow scaled with rarity (brick -> glass) and exotic variety palettes.
  function cutView(canvas, seed, quality, variety, opts) {
    opts = opts || {};
    const dpr = window.devicePixelRatio || 1;
    const w = canvas.clientWidth || 300, h = canvas.clientHeight || 220;
    canvas.width = w * dpr; canvas.height = h * dpr;
    const ctx = canvas.getContext('2d');
    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, w, h);

    const qKey = quality || 'bean';
    // 隱藏彩蛋：不是玉石就畫寶石（外殼石皮 + 切面寶石 + 閃光）
    if (GEM_STYLE[qKey]) {
      const halo2 = ctx.createRadialGradient(w / 2, h / 2, 4, w / 2, h / 2, Math.min(w, h) * 0.5);
      halo2.addColorStop(0, 'rgba(255,255,255,0.16)');
      halo2.addColorStop(1, 'rgba(255,255,255,0)');
      ctx.fillStyle = halo2;
      ctx.fillRect(0, 0, w, h);
      gemCutView(ctx, seed, w, h, qKey, opts.grade);
      return;
    }

    // rarity tiers 0..4 (brick..glass)
    const rarity = { brick: 0, bean: 1, oilgreen: 2, icy: 3, glass: 4 }[qKey] != null
      ? { brick: 0, bean: 1, oilgreen: 2, icy: 3, glass: 4 }[qKey] : 1;
    const seedNum = rngFrom(seed);

    // backdrop halo sized by rarity
    const halo = ctx.createRadialGradient(w/2, h/2, 4, w/2, h/2, Math.min(w,h)*0.48);
    halo.addColorStop(0, `rgba(255,240,180,${0.10 + rarity * 0.10})`);
    halo.addColorStop(1, 'rgba(255,240,180,0)');
    ctx.fillStyle = halo;
    ctx.fillRect(0, 0, w, h);

    // stone body: blob on left half = skin, right half = interior
    const cx = w * 0.5, cy = h * 0.5, rad = Math.min(w, h) * 0.40;
    const pts = stoneBlob(rngFrom(seed), cx, cy, rad);

    // interior fills the whole blob (revealed by the cut)
    drawInterior(ctx, seed, w, h, { quality: qKey, variety: variety || 'base' });

    // skin remains only on the LEFT half (as if knife entered from top)
    ctx.save();
    pathPts(ctx, pts);
    ctx.clip();
    const half = cx - rad * 0.15;
    ctx.save();
    ctx.beginPath();
    ctx.rect(0, 0, half, h);
    ctx.clip();
    drawSkin(ctx, seed, opts.grade != null ? opts.grade : 0, w, h);
    ctx.restore();

    // cut plane line — brighter with rarity
    ctx.strokeStyle = `rgba(255,255,255,${0.25 + rarity * 0.15})`;
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    ctx.moveTo(half, 0);
    ctx.lineTo(half, h);
    ctx.stroke();

    // flash for icy/glass
    if (qKey === 'icy' || qKey === 'glass') {
      const sg = ctx.createLinearGradient(cx - rad, cy - rad, cx + rad, cy + rad);
      sg.addColorStop(0, 'rgba(255,255,255,0)');
      sg.addColorStop(0.55, `rgba(255,255,255,${qKey === 'glass' ? 0.18 : 0.10})`);
      sg.addColorStop(1, 'rgba(255,255,255,0)');
      ctx.fillStyle = sg;
      ctx.fillRect(0, 0, half, h);
    }
    ctx.restore();

    // sparkle particles on high rarity
    if (rarity >= 3) {
      const n = 14 + rarity * 8;
      for (let i = 0; i < n; i++) {
        const r = rngFrom(seed + ':' + i);
        const ang = r() * Math.PI * 2;
        const dist = rad * (0.55 + r() * 0.5);
        const x = cx + Math.cos(ang) * dist * 0.5 + rad * 0.1;
        const y = cy + Math.sin(ang) * dist * 0.5;
        ctx.fillStyle = `rgba(255,255,240,${0.3 + r() * 0.5})`;
        const s = 1 + r() * 2;
        ctx.beginPath();
        ctx.arc(x, y, s, 0, Math.PI * 2);
        ctx.fill();
      }
    }
    return { rarity };
  }

  global.StoneRender = {
    rngFrom, thumbnail, revealView, drawSkin, drawInterior,
    drawCracks, drawLightCone, VARIETY_PALETTE, ScratchBoard, cutView,
  };
})(window);