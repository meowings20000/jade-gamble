// app.js — Jade Gamble SPA (vanilla JS, no build step).
'use strict';

const $ = (sel) => document.querySelector(sel);
const fmt = (n) => (typeof n === 'number' ? n : Number(n) || 0).toLocaleString('zh-TW');

let me = null;

async function api(method, path, body) {
  const opt = { method, headers: { 'Content-Type': 'application/json' } };
  if (body !== undefined) opt.body = JSON.stringify(body);
  const resp = await fetch(path, opt);
  let data = null;
  try { data = await resp.json(); } catch { /* empty body */ }
  if (!resp.ok) {
    const msg = (data && data.error) || `HTTP ${resp.status}`;
    throw new Error(msg);
  }
  return data;
}

function toast(msg, isGold) {
  const el = document.createElement('div');
  el.className = 'toast';
  if (isGold) el.style.borderColor = 'var(--gold)';
  el.textContent = msg;
  document.body.appendChild(el);
  setTimeout(() => el.remove(), 2600);
}

function setChips(n) {
  $('#chips').textContent = fmt(n) + ' 籌碼';
  if (me) me.chips = n;
}

// ---------- views ----------
function show(view) {
  document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
  $('#view-' + view).classList.add('active');
  document.querySelectorAll('nav button').forEach(b =>
    b.classList.toggle('active', b.dataset.view === view));
}

document.querySelectorAll('nav button').forEach(b =>
  b.addEventListener('click', () => {
    show(b.dataset.view);
    ({ shop: loadShop, warehouse: loadWarehouse, market: loadMarket,
       exchange: loadExchange, collection: loadCollection, ranks: loadRanks })[b.dataset.view]();
  }));


// ---------- 繁簡切換 ----------
const I18N = {
  'zh-TW': {
    'shop': '商店', 'warehouse': '倉庫', 'market': '競標場', 'exchange': '兌換所',
    'collection': '圖鑑', 'ranks': '排行榜', 'logout': '登出', 'classic': '傳統模式',
    'yboss': 'Y佬模式', 'langBtn': '简',
  },
  'zh-CN': {
    'shop': '商店', 'warehouse': '仓库', 'market': '竞标场', 'exchange': '兑换所',
    'collection': '图鉴', 'ranks': '排行榜', 'logout': '登出', 'classic': '传统模式',
    'yboss': 'Y佬模式', 'langBtn': '繁',
  },
};
let LANG = localStorage.getItem('lang') || 'zh-TW';
function setLang(lang) {
  LANG = lang;
  localStorage.setItem('lang', lang);
  const t = I18N[lang];
  document.querySelectorAll('nav button[data-view]').forEach(b => { b.textContent = t[b.dataset.view]; });
  const classicBtn = document.getElementById('classic-bet');
  if (classicBtn) classicBtn.textContent = t.classic;
  const yb = document.getElementById('yboss-bet');
  if (yb) yb.textContent = t.yboss;
  const lo = document.getElementById('logout');
  if (lo) lo.textContent = t.logout;
  const lb = document.getElementById('lang-toggle');
  if (lb) lb.textContent = I18N[lang === 'zh-TW' ? 'zh-CN' : 'zh-TW'].langBtn;
}

// On narrow screens the header can't fit both mode buttons: move them into
// the scrollable nav strip (mobile bottom-sheet style) via CSS class only.
const modeHome = null; // original parent captured lazily
function layoutMobileModes() {
  const narrow = window.matchMedia('(max-width: 640px)').matches;
  document.body.classList.toggle('mobile', narrow);
  const nav = document.getElementById('nav');
  const classic = document.getElementById('classic-bet');
  const yboss = document.getElementById('yboss-bet');
  const balance = document.getElementById('balance');
  if (!nav || !classic || !yboss || !balance) return;
  if (narrow && classic.parentElement !== nav) {
    nav.appendChild(classic);   // order: shop..ranks, classic, yboss
    nav.appendChild(yboss);
  } else if (!narrow && classic.parentElement === nav) {
    balance.insertBefore(yboss, document.getElementById('lang-toggle'));
    balance.insertBefore(classic, yboss);
  }
}
window.addEventListener('resize', layoutMobileModes);
function initLangToggle() {
  const btn = document.getElementById('lang-toggle');
  if (!btn) return;
  btn.addEventListener('click', () => setLang(LANG === 'zh-TW' ? 'zh-CN' : 'zh-TW'));
  setLang(LANG);
  layoutMobileModes();
}

// ---------- auth ----------
async function refreshMe() {
  try { me = await api('GET', '/api/me'); } catch { me = null; }
  const logged = !!me;
  $('#nav').style.display = logged ? 'flex' : 'none';
  $('#logout').style.display = logged ? '' : 'none';
  $('#classic-bet').style.display = logged ? '' : 'none';
  $('#yboss-bet').style.display = logged ? '' : 'none';
  layoutMobileModes();
  if (logged) {
    setChips(me.chips);
    $('#userbox').textContent = me.username + (me.title ? ` · ${me.title}` : '');
    show('shop');
    loadShop();
  } else {
    show('login');
    api('GET', '/api/status').then(st => {
      $('#login-discord').style.display = st.discord_oauth ? '' : 'none';
      if (st.mock_auth) $('#login-mock').style.display = '';
    }).catch(() => {});
  }
}

$('#login-discord').addEventListener('click', () => location.href = '/api/auth/discord');
$('#login-mock').addEventListener('click', async () => {
  try {
    await api('POST', '/api/auth/mock', { username: '訪客' + Math.floor(Math.random() * 10000) });
    refreshMe();
  } catch (e) { toast(e.message); }
});
$('#logout').addEventListener('click', async () => {
  await api('POST', '/api/auth/logout');
  me = null;
  location.reload();
});

// ---------- shop ----------
let shopData = null;

async function loadShop() {
  if (!me) return;
  shopData = await api('GET', '/api/shop');
  setChips(shopData.chips);
  const wrap = $('#shop-grades');
  wrap.innerHTML = '';
  for (const g of shopData.grades) {
    const sec = document.createElement('div');
    sec.className = 'card shelf';
    const head = document.createElement('div');
    head.className = 'shelf-head';
    head.innerHTML = `<h3>${g.name}</h3>
      <div class="row">
        <span class="price-note">下次刷新 ${fmt(g.next_refresh)} 籌碼</span>
        <button class="btn ghost" data-refresh="${g.grade}">刷新貨架</button>
      </div>`;
    sec.appendChild(head);
    const grid = document.createElement('div');
    grid.className = 'grid';
    grid.style.gridTemplateColumns = 'repeat(auto-fill,minmax(170px,1fr))';
    for (const item of g.items) {
      grid.appendChild(stoneCardEl(item, g.grade));
    }
    sec.appendChild(grid);
    wrap.appendChild(sec);
    head.querySelector('[data-refresh]').addEventListener('click', async (ev) => {
      try {
        const res = await api('POST', '/api/shop/refresh', { grade: g.grade });
        toast('貨架已刷新');
        shopData = res; setChips(res.chips);
        // re-render
        loadShop();
      } catch (e) { toast(e.message); }
    });
  }
}

function stoneCardEl(item, grade) {
  const card = document.createElement('div');
  card.className = 'stone-card';
  const cv = document.createElement('canvas');
  card.appendChild(cv);
  const price = document.createElement('div');
  price.className = 'price';
  price.textContent = fmt(item.price) + ' 籌碼';
  card.appendChild(price);
  const hint = document.createElement('div');
  hint.className = 'hint';
  hint.textContent = item.hint || '';
  card.appendChild(hint);
  if (item.window_desc) {
    const wd = document.createElement('div');
    wd.className = 'meta';
    wd.textContent = '🪟 ' + item.window_desc;
    card.appendChild(wd);
  }
  const row = document.createElement('div');
  row.className = 'row';
  if (grade >= 1) {
    const light = document.createElement('button');
    light.className = 'btn ghost';
    light.textContent = '打燈 ' + fmt(Math.round(item.price / 20));
    light.addEventListener('click', async () => {
      try {
        const res = await api('POST', '/api/shop/light', { stone_id: item.id });
        hint.textContent = res.report;
        setChips(res.chips);
        toast('💡 ' + res.report);
      } catch (e) { toast(e.message); }
    });
    row.appendChild(light);
  }
  const buy = document.createElement('button');
  buy.className = 'btn';
  buy.textContent = '買下';
  buy.addEventListener('click', async () => {
    try {
      const res = await api('POST', '/api/shop/buy', { stone_id: item.id });
      setChips(res.chips);
      toast('已入倉庫');
      card.remove();
    } catch (e) { toast(e.message); }
  });
  row.appendChild(buy);
  card.appendChild(row);
  // paint thumbnail async
  requestAnimationFrame(() => StoneRender.thumbnail(cv, item.seed, grade));
  return card;
}

// ---------- warehouse ----------
async function loadWarehouse() {
  if (!me) return;
  const inv = await api('GET', '/api/inventory');
  const wrap = $('#warehouse-list');
  wrap.innerHTML = '';
  if (!inv.stones.length) {
    wrap.innerHTML = '<p style="color:var(--muted);font-size:13px">倉庫空空如也——去商店挑顆石頭吧。</p>';
    return;
  }
  for (const st of inv.stones) {
    const card = document.createElement('div');
    card.className = 'stone-card';
    if (st.origin === 'classic') card.style.borderColor = 'var(--gold)';
    const cv = document.createElement('canvas');
    card.appendChild(cv);
    const meta = document.createElement('div');
    meta.className = 'meta';
    meta.textContent = (st.origin === 'classic' ? '傳統石 ' : '') +
      ['公斤料', '表現料', '開窗料'][st.grade] + ' · ' + fmt(st.price) + ' 籌碼';
    card.appendChild(meta);
    const row = document.createElement('div');
    row.className = 'row';
    for (const act of [['切', () => doCut(st)], ['刮', () => doScratch(st)], ['磨', () => doPolish(st)]]) {
      const b = document.createElement('button');
      b.className = 'btn';
      b.textContent = act[0];
      b.addEventListener('click', act[1]);
      row.appendChild(b);
    }
    if (st.origin !== 'classic') {
      const list = document.createElement('button');
      list.className = 'btn ghost';
      list.textContent = '掛賣';
      list.addEventListener('click', () => doList(st));
      row.appendChild(list);
    } else {
      const tag = document.createElement('span');
      tag.className = 'badge';
      tag.style.alignSelf = 'center';
      tag.textContent = '只能切/磨/刮';
      row.appendChild(tag);
    }
    card.appendChild(row);
    wrap.appendChild(card);
    requestAnimationFrame(() => StoneRender.thumbnail(cv, st.seed, st.grade));
  }
}

// ---------- cut ----------
async function doCut(st) {
  if (!confirm('一刀下去就不能回頭。要切嗎？')) return;
  const res = await api('POST', '/api/cut', { stone_id: st.id });
  setChips(res.chips);
  showResultModal('切石', res, st);
  loadWarehouse();
}

function showResultModal(title, res, st) {
  const bg = document.createElement('div');
  bg.className = 'modal-bg';
  const m = document.createElement('div');
  m.className = 'modal';
  const win = res.payout > (st ? st.price : 0);
  const eggHTML = res.egg === 'bianhe'
    ? '<div class="egg-banner">卞和之石！神仙難斷寸玉，而你賭贏了傳說。</div>'
    : (res.egg === 'b_fake' ? '<div class="egg-banner">B貨騙局——皮殼表現全是偽裝，酸洗注膠。</div>' : '');
  const showCutView = res.quality && ['砖头料','豆种','油青种','冰种','玻璃种'].includes(res.quality) && res.variety && res.variety !== '-';
  m.innerHTML = `
    <h3>${title}結果</h3>
    ${eggHTML}
    ${showCutView ? '<canvas id="cut-cv"></canvas>' : ''}
    <div class="big-result ${win ? 'win' : 'lose'}">${win ? '+' : ''}${fmt(res.payout)} 籌碼</div>
    <div class="kv"><span>品質</span><b>${res.quality}</b></div>
    <div class="kv"><span>異色</span><b>${res.variety}</b></div>
    <div class="kv"><span>倍率</span><b>×${res.multiplier || (res.mult || '-')}</b></div>
    ${res.first_discovery ? `<div class="egg-banner">🆕 圖鑑新發現：${res.variety}（收藏分 +${res.collection_gain}）</div>` : ''}
    ${res.title_awarded ? `<div class="egg-banner">🏅 獲得稱號：${res.title_awarded}</div>` : ''}
    ${res.broke_at !== undefined ? `<div class="kv"><span>磨崩於第</span><b>${res.broke_at} 層</b></div>` : ''}
    ${res.insurance_refund ? `<div class="kv"><span>保險理賠</span><b>+${fmt(res.insurance_refund)}</b></div>` : ''}
    <div class="row" style="margin-top:14px"><button class="btn" id="m-close">收下</button></div>`;
  bg.appendChild(m);
  document.body.appendChild(bg);
  if (showCutView) {
    const cv = m.querySelector('#cut-cv');
    StoneRender.cutView(cv, st ? st.seed : res.seed, qualityKey(res.quality), varietyKey(res.variety), { grade: st ? st.grade : 0 });
  }
  m.querySelector('#m-close').addEventListener('click', () => bg.remove());
}

// ---------- scratch ----------
async function doScratch(st) {
  const res = await api('POST', '/api/scratch/start', { stone_id: st.id });
  const bg = document.createElement('div');
  bg.className = 'modal-bg';
  const m = document.createElement('div');
  m.className = 'modal';
  m.innerHTML = `
    <h3>刮石 <span style="font-size:12px;color:var(--muted)">${st.id}</span></h3>
    <div class="scratch-board"><canvas id="scr-cv"></canvas></div>
    <div class="kv"><span>已累積</span><b id="scr-acc">0</b></div>
    <p style="font-size:12px;color:var(--muted);margin:8px 0">
      按住拖动像真刮石一样刮开石皮；轻点一格可快速刮。
      刮到裂紋會跌價，全開 12 格有 1.08× 獎勵。</p>
    <div class="row" style="margin-top:10px">
      <button class="btn ghost" id="scr-hint">看提示</button>
      <button class="btn" id="scr-sell">見好就收（-4%）</button>
      <button class="btn ghost" id="scr-close">離開（下次繼續）</button>
    </div>`;
  bg.appendChild(m);
  document.body.appendChild(bg);

  const cv = m.querySelector('#scr-cv');
  const accEl = m.querySelector('#scr-acc');
  const board = new StoneRender.ScratchBoard(cv, st.seed, st.grade, {
    onCellReveal: async (cell) => {
      try {
        const rr = await api('POST', '/api/scratch/reveal', { stone_id: st.id, cell });
        board.markRevealed(cell, rr.kind);
        accEl.textContent = fmt(rr.accumulated);
        if (rr.kind === 'crack' || rr.kind === 'deep_crack') {
          toast(rr.kind === 'deep_crack' ? '💥 深裂！' : '⚠️ 刮到裂紋');
        }
        if (rr.done) {
          setTimeout(() => {
            board.finale(qualityKey(rr.quality), varietyKey(rr.variety));
            setTimeout(() => { bg.remove(); showResultModal('刮石', rr, st); loadWarehouse(); refreshMe(); }, 700);
          }, 350);
        }
      } catch (e) { toast(e.message); }
    },
  });

  // repaint a resumed session
  for (const rv of (res.revealed || [])) {
    if (rv && typeof rv === 'object' && rv.cell !== undefined) {
      board.markRevealed(rv.cell, rv.kind);
    }
  }
  accEl.textContent = fmt(res.accumulated);

  m.querySelector('#scr-hint').addEventListener('click', async () => {
    const un = [];
    for (let i = 0; i < 12; i++) {
      if (!board.revealed.has(i)) un.push(i);
    }
    if (!un.length) return;
    const cell = un[Math.floor(Math.random() * un.length)];
    try { const rr = await api('POST', '/api/scratch/hint', { stone_id: st.id, cell }); toast(rr.hint); }
    catch (e) { toast(e.message); }
  });
  m.querySelector('#scr-sell').addEventListener('click', async () => {
    try {
      const rr = await api('POST', '/api/scratch/sell', { stone_id: st.id });
      bg.remove();
      showResultModal('刮石（提前賣出）', rr, st);
      loadWarehouse(); refreshMe();
    } catch (e) { toast(e.message); }
  });
  m.querySelector('#scr-close').addEventListener('click', () => { board.destroy(); bg.remove(); });
}

function qualityKey(name) {
  return { '砖头料': 'brick', '磚頭料': 'brick', '豆种': 'bean', '豆種': 'bean',
    '油青种': 'oilgreen', '油青種': 'oilgreen', '冰种': 'icy', '冰種': 'icy',
    '玻璃种': 'glass', '玻璃種': 'glass' }[name] || '';
}
function varietyKey(name) {
  const m = { '底色': 'base', '紫罗兰': 'violet', '紫羅蘭': 'violet', '蓝水': 'bluewater', '藍水': 'bluewater',
    '白底青': 'whitegreen', '飘蓝花': 'floatingblue', '飄藍花': 'floatingblue',
    '墨翠': 'inkgreen', '黄加绿': 'yellowgreen', '黃加綠': 'yellowgreen',
    '春带彩': 'springpurple', '福禄寿': 'fortune3', '福祿壽': 'fortune3', '帝王绿': 'imperial', '帝王綠': 'imperial' };
  return m[name] || 'base';
}

// ---------- polish ----------
async function doPolish(st) {
  // 開磨前先選力度——種水已經定死，力度配不上就每層賭命。
  const bg = document.createElement('div');
  bg.className = 'modal-bg';
  const m = document.createElement('div');
  m.className = 'modal';
  const pick = (force, name, desc) => `
    <button class="btn ghost" data-force="${force}" style="display:block;width:100%;text-align:left;margin-bottom:8px">
      <b>${name}</b><br><span style="font-size:12px;color:var(--muted)">${desc}</span>
    </button>`;
  m.innerHTML = `
    <h3>磨石 — 選力度</h3>
    <p style="font-size:13px;color:var(--muted);line-height:1.6">
      這顆料的種水在你買下它時就定死了，<b>該用多大力度也跟著定死了</b>。<br>
      力度配得上，機器順暢一路上去；配不上，每一層都在賭命。<br>
      打法燈報告和皮殼表現猜猜看——選了就不能換。</p>
    ${pick(1, '輕磨 ×0.93 起', '最保險，但磨得慢、天花板低。有裂的料只能這樣磨。')}
    ${pick(2, '正磨 ×0.93 起', '標準力度。一般料吃得住。')}
    ${pick(3, '重磨 ×0.93 起', '吃得下重壓的只有好種水。壓不住就崩。')}
    <div class="row" style="margin-top:6px"><button class="btn ghost" id="pol-cancel">離開</button></div>`;
  bg.appendChild(m);
  document.body.appendChild(bg);
  m.querySelector('#pol-cancel').addEventListener('click', () => bg.remove());
  m.querySelectorAll('[data-force]').forEach((btn) => {
    btn.addEventListener('click', () => {
      bg.remove();
      startPolish(st, Number(btn.dataset.force));
    });
  });
}

async function startPolish(st, force) {
  const res = await api('POST', '/api/polish/start', { stone_id: st.id, force });
  const bg = document.createElement('div');
  bg.className = 'modal-bg';
  const m = document.createElement('div');
  m.className = 'modal';
  const pct = (p) => (p * 100).toFixed(1) + '%';
  m.innerHTML = `
    <h3>磨石 — ${res.force_name} <span style="font-size:12px;color:var(--muted)">皮殼一寸寸磨掉</span></h3>
    <div class="big-result" id="pol-mult">×${Number(res.multiplier).toFixed(2)}</div>
    <div class="ladder" id="pol-ladder"></div>
    <div class="kv"><span>下一層爆裂機率</span><b id="pol-risk">${pct(res.break_prob)}</b></div>
    <p id="pol-feel" style="font-size:13px;color:var(--gold);margin:10px 0;line-height:1.6">👁 ${res.feel}</p>
    <p style="font-size:12px;color:var(--muted);margin:8px 0">
      開磨即損 7% 皮殼價。力度配得上就磨得順，配不上每層都在賭命——
      <b>手感會告訴你配不配</b>，隨時可以落袋。</p>
    <div class="row" style="margin-top:10px">
      <button class="btn" id="pol-adv">再磨一層</button>
      <button class="btn danger" id="pol-cash">落袋 ×${Number(res.multiplier).toFixed(2)}</button>
      <button class="btn ghost" id="pol-close">離開</button>
    </div>`;
  bg.appendChild(m);
  document.body.appendChild(bg);
  const ladder = m.querySelector('#pol-ladder');
  const paintLadder = (stage) => {
    ladder.innerHTML = '';
    for (let i = 0; i <= 10; i++) {
      const el = document.createElement('span');
      el.className = 'rung' + (i < stage ? ' past' : '') + (i === stage ? ' cur' : '');
      el.textContent = '×' + (0.93 * Math.pow(1.2, i) > res.ladder.top ? res.ladder.top : (0.93 * Math.pow(1.2, i)).toFixed(2));
      ladder.appendChild(el);
    }
  };
  paintLadder(res.stage);
  m.querySelector('#pol-adv').addEventListener('click', async () => {
    try {
      const rr = await api('POST', '/api/polish/advance', { stone_id: st.id });
      if (rr.alive) {
        m.querySelector('#pol-mult').textContent = '×' + Number(rr.multiplier).toFixed(2);
        m.querySelector('#pol-cash').textContent = '落袋 ×' + Number(rr.multiplier).toFixed(2);
        m.querySelector('#pol-risk').textContent = pct(rr.break_prob);
        m.querySelector('#pol-feel').textContent = '👁 ' + rr.feel;
        paintLadder(rr.stage);
        if (rr.at_top) m.querySelector('#pol-adv').disabled = true;
      } else {
        bg.remove();
        showResultModal('磨石', { ...rr, payout: 0, quality: '已碎', variety: '-', multiplier: Number(rr.multiplier).toFixed(2) }, st);
        loadWarehouse(); refreshMe();
      }
    } catch (e) { toast(e.message); }
  });
  m.querySelector('#pol-cash').addEventListener('click', async () => {
    try {
      const rr = await api('POST', '/api/polish/cash', { stone_id: st.id });
      bg.remove();
      showResultModal('磨石落袋', rr, st);
      loadWarehouse(); refreshMe();
    } catch (e) { toast(e.message); }
  });
  m.querySelector('#pol-close').addEventListener('click', () => bg.remove());
}

// ---------- market ----------
async function doList(st) {
  const price = prompt('掛單價（籌碼）？', String(st.price));
  if (!price) return;
  try {
    const res = await api('POST', '/api/market/list', { stone_id: st.id, ask_price: Number(price) });
    toast(`已掛單（手續費 ${fmt(res.fee)}）`);
    loadWarehouse();
  } catch (e) { toast(e.message); }
}

async function loadMarket() {
  const data = await api('GET', '/api/market');
  const wrap = $('#market-list');
  wrap.innerHTML = '';
  if (!data.listings.length) {
    wrap.innerHTML = '<p style="color:var(--muted);font-size:13px">目前沒有掛單。從倉庫掛一顆試試？</p>';
    return;
  }
  for (const l of data.listings) {
    const card = document.createElement('div');
    card.className = 'stone-card';
    const cv = document.createElement('canvas');
    card.appendChild(cv);
    const price = document.createElement('div');
    price.className = 'price';
    price.textContent = fmt(l.ask_price) + ' 籌碼';
    card.appendChild(price);
    const seller = document.createElement('div');
    seller.className = 'meta';
    seller.textContent = (l.npc ? '⛏ 礦區直送' : '🕶 匿名賣家') + ' · ' + ['公斤料', '表現料', '開窗料'][l.grade];
    card.appendChild(seller);
    const hint = document.createElement('div');
    hint.className = 'hint';
    hint.textContent = l.light_hint || '';
    card.appendChild(hint);
    const buy = document.createElement('button');
    buy.className = 'btn';
    buy.textContent = '買下';
    buy.addEventListener('click', async () => {
      try {
        const res = await api('POST', '/api/market/buy', { listing_id: l.id });
        setChips(res.chips);
        toast('得標！石頭已入倉庫');
        loadMarket();
      } catch (e) { toast(e.message); }
    });
    card.appendChild(buy);
    wrap.appendChild(card);
    requestAnimationFrame(() => StoneRender.thumbnail(cv, l.seed, l.grade));
  }
}

// ---------- exchange ----------
async function loadExchange() {
  const data = await api('GET', '/api/exchange');
  setChips(data.chips);
  const wrap = $('#exchange-list');
  wrap.innerHTML = '';
  for (const it of data.catalog) {
    const card = document.createElement('div');
    card.className = 'card';
    card.innerHTML = `<h3 style="color:var(--gold);font-size:15px">${it.name}
      <span class="badge">${it.kind === 'buff' ? '限時' : it.kind === 'cosmetic' ? '裝飾' : '消耗品'}</span></h3>
      <p style="font-size:13px;color:var(--muted);margin:8px 0">${it.description}</p>
      <button class="btn">${it.price ? fmt(it.price) + ' 籌碼' : '按石頭計價'}</button>`;
    card.querySelector('button').addEventListener('click', async () => {
      try {
        const res = await api('POST', '/api/exchange/buy', { key: it.key });
        if (res.payouts) toast(`刮到爽！十顆共 +${fmt(res.total)} 籌碼`);
        else toast('已兌換');
        refreshMe();
        loadExchange();
      } catch (e) { toast(e.message); }
    });
    wrap.appendChild(card);
  }
}

$('#relief-chips').addEventListener('click', async () => {
  try { await api('POST', '/api/relief', { option: 'chips' }); toast('救濟已入帳'); refreshMe(); }
  catch (e) { toast(e.message); }
});
$('#relief-ticket').addEventListener('click', async () => {
  try { const r = await api('POST', '/api/relief', { option: 'ticket' }); toast(`刮到爽 +${fmt(r.total)} 籌碼`); refreshMe(); }
  catch (e) { toast(e.message); }
});

// ---------- collection ----------
async function loadCollection() {
  const data = await api('GET', '/api/collection');
  $('#coll-score').textContent = '· 收藏分 ' + fmt(data.score);
  const wrap = document.getElementById('collection-list');
  wrap.innerHTML = '';
  for (const v of data.varieties) {
    const card = document.createElement('div');
    card.className = 'card';
    const pal = StoneRender.VARIETY_PALETTE[varietyKey(v.name)] || StoneRender.VARIETY_PALETTE.base;
    card.innerHTML = `<div style="height:56px;border-radius:8px;background:linear-gradient(120deg,${pal[2]},${pal[0]},${pal[1]})"></div>
      <h3 style="font-size:14px;margin-top:8px">${v.discovered ? v.name : '？？？'}</h3>
      <p style="font-size:12px;color:var(--muted)">${v.discovered ? '已發現 · ' + v.score + ' 分' : '未發現'}</p>`;
    wrap.appendChild(card);
  }
}

// ---------- ranks ----------
async function loadRanks() {
  const lb = await api('GET', '/api/leaderboard?kind=wealth');
  const table = $('#rank-table');
  table.innerHTML = '<tr><th>#</th><th>玩家</th><th>籌碼</th></tr>' +
    lb.entries.map((e, i) => `<tr><td class="${i === 0 ? 'rank1' : ''}">${i + 1}</td>
      <td>${e.username}</td><td>${fmt(e.score)}</td></tr>`).join('');
  const hall = await api('GET', '/api/hall');
  $('#hall-table').innerHTML = '<tr><th>玩家</th><th>石頭</th><th>結果</th><th>入帳</th></tr>' +
    (hall.entries.length ? hall.entries.map(e => `<tr><td>${e.username}</td>
      <td>${qualityName(e.quality)}·${varietyName(e.variety)}</td>
      <td>${e.payout >= 0 ? '<span style="color:var(--green)">大漲</span>' : '<span style="color:var(--red)">崩了</span>'}</td>
      <td>${fmt(e.payout)}</td></tr>`).join('')
      : '<tr><td colspan="4" style="color:var(--muted)">還沒有人切出傳說……</td></tr>');
}

function qualityName(q) { return ['砖头料', '豆种', '油青种', '冰种', '玻璃种'][q] || q; }
function varietyName(v) { return ['底色', '紫罗兰', '蓝水', '白底青', '飘蓝花', '墨翠', '黄加绿', '春带彩', '福禄寿', '帝王绿'][v] || v; }

$('#rank-wealth').addEventListener('click', async () => {
  const lb = await api('GET', '/api/leaderboard?kind=wealth');
  $('#rank-table').innerHTML = '<tr><th>#</th><th>玩家</th><th>籌碼</th></tr>' +
    lb.entries.map((e, i) => `<tr><td class="${i === 0 ? 'rank1' : ''}">${i + 1}</td>
      <td>${e.username}</td><td>${fmt(e.score)}</td></tr>`).join('');
  $('#rank-wealth').classList.add('btn'); $('#rank-coll').classList.add('ghost');
});
$('#rank-coll').addEventListener('click', async () => {
  const lb = await api('GET', '/api/leaderboard?kind=collection');
  $('#rank-table').innerHTML = '<tr><th>#</th><th>玩家</th><th>收藏分</th></tr>' +
    lb.entries.map((e, i) => `<tr><td class="${i === 0 ? 'rank1' : ''}">${i + 1}</td>
      <td>${e.username}</td><td>${fmt(e.score)}</td></tr>`).join('');
});

// ---------- 傳統模式 ----------
$('#classic-bet').addEventListener('click', () => {
  const bg = document.createElement('div');
  bg.className = 'modal-bg';
  const m = document.createElement('div');
  m.className = 'modal';
  m.innerHTML = `
    <h3>傳統模式</h3>
    <p style="font-size:13px;color:var(--muted);line-height:1.6">
      壓上賭資，我給你一塊全盲的石頭——沒有打燈、沒有提示、不能轉賣、不能套型。<br>
      你只有三條路：<b>切</b>、<b>磨</b>、<b>刮</b>。</p>
    <div style="margin:12px 0">
      <div class="kv"><span>賭資下限</span><b>100</b></div>
      <div class="kv"><span>賭資上限</span><b>1,000,000</b></div>
      <div class="kv"><span>檔位規則</span><b>&lt;2,000 公斤料檔 · &lt;20,000 表現料檔 · 以上 開窗料檔</b></div>
    </div>
    <div class="row">
      <input id="stake-input" type="number" min="100" max="1000000" step="100" value="1000"
        style="flex:1;background:var(--panel2);border:1px solid var(--line);color:var(--text);padding:10px;border-radius:8px">
      <button class="btn" id="stake-go">壓注開石</button>
    </div>
    <p style="font-size:12px;color:var(--muted);margin-top:8px" id="stake-err"></p>`;
  bg.appendChild(m);
  document.body.appendChild(bg);
  bg.addEventListener('click', (ev) => { if (ev.target === bg) bg.remove(); });
  m.querySelector('#stake-go').addEventListener('click', async () => {
    const stake = Number(m.querySelector('#stake-input').value);
    const err = m.querySelector('#stake-err');
    try {
      const res = await api('POST', '/api/classic/bet', { stake });
      setChips(res.chips);
      bg.remove();
      toast('傳統石已入手——切、磨、刮，選一條路');
      show('warehouse');
      loadWarehouse();
    } catch (e) { err.textContent = e.message; }
  });
});

initLangToggle();

// ---------- Y佬模式 ----------
$('#yboss-bet').addEventListener('click', () => {
  const bg = document.createElement('div');
  bg.className = 'modal-bg';
  const m = document.createElement('div');
  m.className = 'modal';
  m.innerHTML = `
    <h3>Y佬模式</h3>
    <p style="font-size:13px;color:var(--muted);line-height:1.6">
      純粹的賭：壓上資金，選一條路。<br>
      <b>切一刀</b>——一翻兩瞪眼（EV 95%）。<br>
      <b>磨石</b>——六層階梯 ×1.3 → ×7.5，每層稀有度更高、成功機率更低，磨崩歸零，隨時落袋。</p>
    <div class="row" style="margin:10px 0">
      <input id="yb-stake" type="number" min="100" max="1000000" step="100" value="1000"
        style="flex:1;background:var(--panel2);border:1px solid var(--line);color:var(--text);padding:10px;border-radius:8px">
    </div>
    <div class="row">
      <button class="btn" id="yb-cut">切一刀</button>
      <button class="btn danger" id="yb-polish">開始磨石</button>
    </div>
    <div id="yb-stage" style="margin-top:12px"></div>
    <p style="font-size:12px;color:var(--muted);margin-top:8px" id="yb-err"></p>`;
  bg.appendChild(m);
  document.body.appendChild(bg);
  bg.addEventListener('click', (ev) => { if (ev.target === bg) bg.remove(); });
  const err = m.querySelector('#yb-err');
  const stage = m.querySelector('#yb-stage');

  const bet = async (choice) => {
    const stake = Number(m.querySelector('#yb-stake').value);
    try {
      const res = await api('POST', '/api/yboss/bet', { stake, choice });
      setChips(res.chips);
      if (choice === 'cut') {
        stage.innerHTML = `
          <canvas id="yb-cut-cv" style="width:100%;height:200px;border-radius:10px;background:#0d0b09"></canvas>
          <div class="big-result ${res.payout > 0 ? 'win' : 'lose'}">${res.label}<br>${res.payout > 0 ? '+' + fmt(res.payout) : '歸零'}</div>
          <div class="kv"><span>倍率</span><b>×${res.mult}</b></div>`;
        const ybcv = stage.querySelector('#yb-cut-cv');
        if (ybcv && ['磚頭料','豆種','油青種','冰種','玻璃種'].includes(res.label)) {
          StoneRender.cutView(ybcv, String(res.rung != null ? res.rung : (Date.now() % 2**53)), qualityKey(res.label), 'base', { grade: 0 });
        } else if (ybcv) {
          ybcv.remove();
        }
      } else {
        renderLadder(res);
      }
    } catch (e) { err.textContent = e.message; }
  };

  const renderLadder = (res) => {
    const ladder = res.ladder || [];
    stage.innerHTML = `
      <div class="ladder" id="yb-ladder"></div>
      <div class="big-result" id="yb-mult">×${res.mult} <span style="font-size:14px;color:var(--muted)">${res.label}</span></div>
      <div class="row" style="justify-content:center">
        <button class="btn" id="yb-adv">再磨一層</button>
        <button class="btn danger" id="yb-cash">落袋 ${fmt(res.stake * res.mult)}</button>
      </div>`;
    const lad = stage.querySelector('#yb-ladder');
    ladder.forEach((rg, i) => {
      const el = document.createElement('span');
      el.className = 'rung' + (i < res.rung ? ' past' : '') + (i === res.rung ? ' cur' : '');
      el.textContent = '×' + rg.mult + ' ' + rg.label;
      lad.appendChild(el);
    });
    stage.querySelector('#yb-adv').addEventListener('click', async () => {
      try {
        const rr = await api('POST', '/api/yboss/polish', { advance: true });
        if (rr.alive) {
          renderLadder({ ...res, rung: rr.rung, mult: rr.mult, label: rr.label, ladder });
        } else {
          stage.innerHTML = `<div class="big-result lose">磨崩了！<br>${res.stake > 0 ? '-' + fmt(res.stake) : ''}</div>`;
          refreshMe();
        }
      } catch (e) { err.textContent = e.message; }
    });
    stage.querySelector('#yb-cash').addEventListener('click', async () => {
      try {
        const rr = await api('POST', '/api/yboss/polish', { advance: false });
        setChips(rr.chips);
        stage.innerHTML = `<div class="big-result win">落袋 +${fmt(rr.payout)}</div>
          <div class="kv"><span>層數</span><b>${rr.label} ×${rr.mult}</b></div>`;
        refreshMe();
      } catch (e) { err.textContent = e.message; }
    });
  };

  m.querySelector('#yb-cut').addEventListener('click', () => bet('cut'));
  m.querySelector('#yb-polish').addEventListener('click', () => bet('polish'));
});

refreshMe();