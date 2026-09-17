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

// esc: 使用者輸入（名稱、條件）要 escape 才能進 innerHTML。
function esc(s) {
  return String(s == null ? '' : s).replace(/[&<>"']/g, (c) =>
    ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

function setChips(n) {
  $('#chips').textContent = fmt(n) + ' 喵喵幣';
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
       exchange: () => { loadExchange(); if (typeof loadRewards === 'function') loadRewards(); },
       collection: () => { loadCollection(); loadGemBook(); }, ranks: loadRanks,
       transfer: loadTransfers, admin: loadAdmin, history: loadHistory, bank: loadBank,
       heist: (typeof loadHeist === 'function' ? (() => { loadHeist(); if (typeof heistEnsurePoll === 'function') heistEnsurePoll(); if (!window.__heistTick) window.__heistTick = setInterval(() => { const v = document.getElementById('view-heist'); if (v && v.classList.contains('active') && typeof loadHeist === 'function') { loadHeist(); if (typeof heistSyncChips === 'function') heistSyncChips(); } }, 2500); }) : loadShop) })[b.dataset.view]();
  }));


// ---------- 繁簡切換 ----------
const I18N = {
  'zh-TW': {
    'shop': '商店', 'warehouse': '倉庫', 'market': '競標場', 'exchange': '兌換所',
    'collection': '圖鑑', 'ranks': '排行榜', 'transfer': '轉賬', 'admin': '控制臺', 'history': '紀錄', 'bank': '喵喵錢莊', 'logout': '登出', 'classic': '傳統模式',
    'yboss': 'Y佬模式', 'heist': '奪寶', 'langBtn': '简',
  },
  'zh-CN': {
    'shop': '商店', 'warehouse': '仓库', 'market': '竞标场', 'exchange': '兑换所',
    'collection': '图鉴', 'ranks': '排行榜', 'transfer': '转账', 'admin': '控制台', 'history': '记录', 'bank': '喵喵钱庄', 'logout': '登出', 'classic': '传统模式',
    'yboss': 'Y佬模式', 'heist': '夺宝', 'langBtn': '繁',
  },
};
let LANG = localStorage.getItem('lang') || 'zh-TW';
function setLang(lang) {
  LANG = lang;
  localStorage.setItem('lang', lang);
  const t = I18N[lang];
  // 字典沒收錄（例如瀏覽器快取了舊 JS）就用按鈕原本的文字，永不變空白
  document.querySelectorAll('nav button[data-view]').forEach(b => {
    const label = t[b.dataset.view] || b.dataset.label || b.textContent.trim();
    if (label) b.dataset.label = label;
    b.textContent = label || b.dataset.view;
  });
  const classicBtn = document.getElementById('classic-bet');
  if (classicBtn) classicBtn.textContent = t.classic;
  const yb = document.getElementById('yboss-bet');
  if (yb) yb.textContent = t.yboss;
  const lo = document.getElementById('logout');
  if (lo) lo.textContent = t.logout;
  const tf = document.querySelector('nav button[data-view="transfer"]');
  if (tf && !tf.textContent.trim()) tf.textContent = t.transfer || '轉賬';
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
    // DC 頭像直接讀（OAuth 時存下來的 CDN 連結）＋兌換所的頭像框
    $('#userbox').innerHTML =
      `<span class="avatar-ring ${me.frame ? esc(me.frame) : ''}">` +
      (me.avatar ? `<img src="${esc(me.avatar)}" alt="" referrerpolicy="no-referrer">` : '<span class="ph">🐾</span>') +
      `</span><span>${(me.title ? `【${esc(me.title)}】` : '') + esc(me.username)}</span>`;
    // 管理員才看得到控制臺
    const navAdmin = $('#nav-admin');
    if (navAdmin) navAdmin.style.display = me.is_admin ? '' : 'none';
    // 只有在還沒進任何頁面（或還在登入頁）才切到商店；
    // 否則在奪寶等頁面按到任何觸發 refreshMe 的動作都會被彈回主畫面。
    const __act = document.querySelector('div.view.active');
    if (!__act || __act.id === 'view-login') show('shop');
    loadShop();
    loadEvents();
  } else {
    show('login');
    api('GET', '/api/status').then(st => {
      // 兩顆按鈕預設都在同一頁顯示，只有後端說沒開才收起來
      // （以前是先隱藏再顯示，慢一步看起來就像「要點一下才變按鈕」）
      $('#login-discord').style.display = st.discord_oauth ? '' : 'none';
      $('#login-mock').style.display = st.mock_auth ? '' : 'none';
    }).catch(() => {});
  }
}

$('#login-discord').addEventListener('click', () => location.href = '/api/auth/discord');

// 登入失敗時（?login_error=xxx）在登入頁說清楚原因，而不是丟一個空白頁。
(function reportLoginError() {
  const q = new URLSearchParams(location.search);
  const code = q.get('login_error');
  if (!code) return;
  const MSG = {
    guild: '這個 Discord 帳號不在授權的伺服器內（要加入猪猪岛才能玩）',
    denied: '你在 Discord 按了拒絕授權',
    state: '登入連結過期了，再按一次「用 Discord 登入」就好',
    nocode: 'Discord 沒有回傳授權碼，再試一次',
    token: 'Discord 授權交換失敗，稍後再試',
    user: '讀不到你的 Discord 資料，稍後再試',
    guildcheck: '查不到你的伺服器清單，稍後再試',
    disabled: '這個站台沒有開啟 Discord 登入',
  };
  setTimeout(() => toast(MSG[code] || ('登入失敗：' + code)), 400);
  history.replaceState(null, '', location.pathname);
})();
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
        <span class="price-note">下次刷新 ${fmt(g.next_refresh)} 喵喵幣</span>
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
  price.textContent = fmt(item.price) + ' 喵喵幣';
  card.appendChild(price);
  const hint = document.createElement('div');
  hint.className = 'hint';
  hint.textContent = item.hint || '';
  // 蒙頭料沒打燈就什麼描述都沒有（本來就是全盲的）
  hint.style.display = item.hint ? '' : 'none';
  card.appendChild(hint);
  const row = document.createElement('div');
  row.className = 'row';
  // 三檔都能打燈：蒙頭→模糊、表現→中等、開窗→準確
  const light = document.createElement('button');
  light.className = 'btn ghost';
  light.textContent = '💡 打燈 ' + fmt(Math.round(item.price / 20));
  light.addEventListener('click', async () => {
    try {
      const res = await api('POST', '/api/shop/light', { stone_id: item.id });
      hint.textContent = res.report;
      hint.style.display = '';
      setChips(res.chips);
      toast('💡 ' + res.report);
    } catch (e) { toast(e.message); }
  });
  row.appendChild(light);
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
      ['公斤料', '表現料', '開窗料'][st.grade] + ' · ' + fmt(st.price) + ' 喵喵幣';
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
  // 磨石：落袋是 BaseValue × 倍率，成本要用「買入價」比；兩者不同，不能拿身價直接比
  const cost = res.stone_price !== undefined ? Number(res.stone_price) : (st ? st.price : 0);
  const net = res.net !== undefined ? Number(res.net) : (res.payout - cost);
  const win = net >= 0;
  const gemKey = res.gem || ''; // 宣告必須在使用之前（原本寫在下面 → TDZ 錯誤）
  const eggHTML = gemKey
    ? `<div class="egg-banner">💎 彩蛋！這一刀切出來的不是玉——是 <b>${esc(res.gem_name || gemKey)}</b>！</div>`
    : (res.egg === 'bianhe'
    ? '<div class="egg-banner">卞和之石！神仙難斷寸玉，而你賭贏了傳說。</div>'
    : (res.egg === 'b_fake' ? '<div class="egg-banner">B貨騙局——皮殼表現全是偽裝，酸洗注膠。</div>' : ''));
  const showCutView = !!gemKey || (res.quality && ['砖头料','豆种','油青种','冰种','玻璃种'].includes(res.quality) && res.variety && res.variety !== '-');
  m.innerHTML = `
    <h3>${title}結果</h3>
    ${eggHTML}
    ${showCutView ? '<canvas id="cut-cv"></canvas>' : ''}
    <div class="big-result ${res.polish ? '' : (win ? 'win' : 'lose')}">${res.polish ? '' : (win ? '+' : '')}${fmt(res.payout)} 喵喵幣</div>
    ${res.polish ? `<div class="kv"><span>成本（買入價）</span><b>${fmt(cost)}</b></div>
    <div class="kv"><span>${net >= 0 ? '淨賺' : '淨賠'}</span><b style="color:${net >= 0 ? 'var(--green)' : 'var(--red)'}">${net >= 0 ? '+' : ''}${fmt(net)}</b></div>` : ''}
    <div class="kv"><span>${gemKey ? '寶石' : '品質'}</span><b>${gemKey ? esc(res.gem_name || gemKey) : res.quality}</b></div>
    ${gemKey ? '<div class="kv"><span>材質</span><b>不是玉石</b></div>' : `<div class="kv"><span>異色</span><b>${res.variety}</b></div>`}
    <div class="kv"><span>倍率</span><b>×${res.multiplier || (res.mult || '-')}</b></div>
    ${res.first_discovery ? `<div class="egg-banner">🆕 圖鑑新發現：${res.variety}（收藏分 +${res.collection_gain}）</div>` : ''}
    ${res.title_awarded ? `<div class="egg-banner">🏅 獲得稱號：${res.title_awarded}</div>` : ''}
    ${res.broke_at !== undefined ? `<div class="kv"><span>磨崩於第</span><b>${res.broke_at} 層</b></div>` : ''}
    ${res.salvage ? `<div class="kv"><span>磨崩救回（當前倍率 30%）</span><b style="color:var(--gold)">+${fmt(res.salvage)}</b></div>` : ''}
    ${res.insurance_refund ? `<div class="kv"><span>保險理賠</span><b>+${fmt(res.insurance_refund)}</b></div>` : ''}
    <div class="row" style="margin-top:14px"><button class="btn" id="m-close">收下</button></div>`;
  bg.appendChild(m);
  document.body.appendChild(bg);
  if (showCutView) {
    const cv = m.querySelector('#cut-cv');
    StoneRender.cutView(cv, st ? st.seed : res.seed, gemKey || qualityKey(res.quality), gemKey ? 'base' : varietyKey(res.variety), { grade: st ? st.grade : 0 });
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
  // 已經在磨的石頭：直接續磨（後端會回傳目前層數），不要重選力度
  try {
    await startPolish(st);
    return;
  } catch (e) {
    // 還沒開磨（或這顆不是進行中的那顆）→ 往下走
  }
  // 一次只能磨一顆：如果磨的是「別顆」，給他一條活路直接把那顆結算掉
  try {
    const me2 = await api('GET', '/api/me');
    const other = me2.polish_stone || '';
    if (Number(me2.polish_running || 0) > 0 && other && other !== st.id) {
      const bg2 = document.createElement('div');
      bg2.className = 'modal-bg';
      const m2 = document.createElement('div');
      m2.className = 'modal';
      m2.innerHTML = '<h3>還有一顆在磨</h3>' +
        '<p style="font-size:13px;color:var(--muted);line-height:1.6">一次只能磨一顆喵。<br>' +
        '先把還在磨的那顆結算（落袋），就能磨這顆了。</p>' +
        '<div class="row" style="margin-top:10px">' +
        '<button class="btn" id="pol-cash-other">結算那一顆</button>' +
        '<button class="btn ghost" id="pol-goto">我自己去處理</button></div>';
      bg2.appendChild(m2);
      document.body.appendChild(bg2);
      m2.querySelector('#pol-goto').addEventListener('click', () => bg2.remove());
      m2.querySelector('#pol-cash-other').addEventListener('click', async () => {
        try {
          let got = 0;
          for (const sid of others) {
            try { const r = await api('POST', '/api/polish/cash', { stone_id: sid }); if (r && r.payout) got += r.payout; } catch (e) { /* 單顆失敗不擋其他 */ }
          }
          bg2.remove();
          toast(got ? ('已結算所有在磨的石頭，共 ' + fmt(got) + ' 喵喵幣') : '已結算');
          try { await refreshMe(); } catch (e) {}
          doPolish(st);
        } catch (e) { toast(e.message || String(e)); }
      });
      return;
    }
  } catch (e) { /* 拿不到狀態就別擋 */ }
  doPolishForcePicker(st);
}

// 選力度面板（開磨前、或還沒磨過一層想換力度時）
function doPolishForcePicker(st) {
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
    ${pick(1, '輕磨', '「這料吃不消重手」——<b>磚頭料</b>，或<b>有裂紋</b>的料（每條裂把理想力度往下拉半級、深裂再拉一級）。選對了每層爆裂率最低（磚頭料 27.5%）。')}
    ${pick(2, '正磨', '<b>豆種、油青種</b>這種沒裂的標準料。不確定要選哪個，先選它最不容易大錯。')}
    ${pick(3, '重磨', '只有<b>冰種、玻璃種</b>這種沒裂的好料壓得住（選對時爆裂率最低，玻璃種 20.3%）。壓在差料或有裂的料上＝每層都在賭命。')}
    <div class="card" style="font-size:13px;line-height:1.7;margin-top:10px">
      <div style="color:var(--gold);font-weight:700;margin-bottom:4px">三種力度到底差在哪</div>
      • 開磨成本<b>三種都一樣</b>（×0.93），差別只在<b>每層的爆裂率</b>。<br>
      • <b>選對力度</b>＝爆裂率最低：磚頭料 27.5%／豆種 25.4%／油青 23.5%／冰種 21.8%／玻璃種 20.3%。<br>
      • <b>選錯一級 +18%</b>、錯兩級 +36%（可以在賭命）。<br>
      • 每層 <b>×1.22</b>；天花板由<b>種水</b>決定，跟力度無關：磚 5.0×／豆 5.5×／油青 6.0×／冰 7.0×／玻璃 8.0×。<br>
      • 磨崩<b>不會歸零</b>：救回當前倍率的 30%。<br>
      • 手感會誠實告訴你配不配——磨第一層之前不用錢，磨了就不能換力度。
    </div>
    <div class="row" style="margin-top:6px"><button class="btn ghost" id="pol-cancel">離開</button></div>`;
  bg.appendChild(m);
  document.body.appendChild(bg);
  m.querySelector('#pol-cancel').addEventListener('click', () => bg.remove());
  m.querySelectorAll('[data-force]').forEach((btn) => {
    btn.addEventListener('click', () => {
      bg.remove();
      startPolish(st, Number(btn.dataset.force)).catch((e) => toast(e.message || String(e)));
    });
  });
}

async function startPolish(st, force) {
  const body = { stone_id: st.id };
  if (force) body.force = force; // 不帶＝續磨既有的一輪
  const res = await api('POST', '/api/polish/start', body);
  const bg = document.createElement('div');
  bg.className = 'modal-bg';
  const m = document.createElement('div');
  m.className = 'modal';
  const pct = (p) => (p * 100).toFixed(1) + '%';
  m.innerHTML = `
    <h3>磨石 — ${res.force_name} <span style="font-size:12px;color:var(--muted)">皮殼一寸寸磨掉</span></h3>
    <div class="big-result" id="pol-mult">×${Number(res.multiplier).toFixed(2)}</div>
    <div class="kv"><span>目前倍率</span><b id="pol-mult2">×${Number(res.multiplier).toFixed(2)}</b></div>
    <div class="kv"><span>這顆料的價值</span><b style="color:var(--muted)">落袋那一刻才知道</b></div>
    <div class="ladder" id="pol-ladder"></div>
    <div class="kv"><span>下一層爆裂機率（第一層最多 25%）</span><b id="pol-risk">${pct(res.break_prob)}</b></div>
    <div class="kv"><span>下一層倍率</span><b id="pol-next"></b></div>
    <p id="pol-feel" style="font-size:13px;color:var(--gold);margin:10px 0;line-height:1.6">👁 ${res.feel}</p>
    <p style="font-size:12px;color:var(--muted);margin:8px 0">
      開磨即損 7% 皮殼價。磨的過程<b>不顯示價值</b>（傳統玩法的規矩），落袋那一刻才結算。力度配得上就磨得順，配不上每層都在賭命——
      <b>手感會告訴你配不配</b>，隨時可以落袋。</p>
    <div class="row" style="margin-top:10px">
      <button class="btn" id="pol-adv">再磨一層</button>
      <button class="btn danger" id="pol-cash">落袋</button>
      ${Number(res.stage) === 0 ? '<button class="btn ghost" id="pol-reforce">換力度（還沒磨過）</button>' : ''}
      <button class="btn ghost" id="pol-close">暫時關掉（進度保留）</button>
    </div>`;
  bg.appendChild(m);
  document.body.appendChild(bg);
  const ladder = m.querySelector('#pol-ladder');
  // 磨石要看到「實際值多少、賺還是賠」——不然不知道自己在賺還是在賠
  // 顯示「伺服器真的會發多少」＝ BaseValue × 倍率（不是用你付的價格算，兩者不同）
  const baseValue = Number(res.base_value || 0) || Math.round(st.price * Number(res.multiplier || 1));
  const price = Number(res.stone_price || st.price || 0);
  // 不顯示價值：磨的過程只看倍率、爆裂率與手感，價值落袋才結算
  const paintValue = (mult) => {
    const mm = m.querySelector('#pol-mult2');
    if (mm) mm.textContent = '×' + Number(mult).toFixed(2);
    const cashBtn = m.querySelector('#pol-cash');
    if (cashBtn) cashBtn.textContent = '落袋（結算才知道價值）';
  };
  paintValue(res.multiplier);
  const paintNext = (mult, top) => {
    const nextMult = Math.min(Number(mult) * 1.25, top || 99);
    const el = m.querySelector('#pol-next');
    if (el) el.innerHTML = `×${nextMult.toFixed(2)}` + (top && nextMult >= top ? '（已到這顆料的天花板）' : '');
  };
  paintNext(res.multiplier, res.ladder && res.ladder.top);
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
        paintValue(rr.multiplier);
        m.querySelector('#pol-risk').textContent = pct(rr.break_prob) + (rr.stage === 0 ? '（第一層上限 25%）' : '');
        m.querySelector('#pol-feel').textContent = '👁 ' + rr.feel;
        const rfBtn = m.querySelector('#pol-reforce');
        if (rfBtn && Number(rr.stage) > 0) rfBtn.remove(); // 磨過一層就不能換力度了
        paintLadder(rr.stage);
        paintNext(rr.multiplier, res.ladder && res.ladder.top);
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
  const pc = m.querySelector('#pol-close'); if (pc) pc.addEventListener('click', () => bg.remove());
  const pf = m.querySelector('#pol-reforce');
  if (pf) pf.addEventListener('click', () => { bg.remove(); doPolishForcePicker(st); });
}

// ---------- market ----------
async function doList(st) {
  const price = prompt('掛單價（喵喵幣）？', String(st.price));
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
  // 拍賣 bot 的收料紀錄（讓玩家知道放久的料有人會收）
  const botBox = $('#market-bots');
  if (botBox) {
    const buys = data.bot_buys || [];
    botBox.innerHTML = buys.length
      ? '🤖 最近收料：' + buys.map((b) =>
          `${b.bot} 收了 ${b.stone_id.slice(0, 8)}（${fmt(b.price)} 喵喵幣）${b.note ? `：「${esc(b.note)}」` : ''}`).join(' · ')
      : '🤖 放太久的料，拍賣場的收料機器人會來接（走漏眼的出價高，精明的只撿便宜）。';
  }
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
    price.textContent = fmt(l.ask_price) + ' 喵喵幣';
    card.appendChild(price);
    const seller = document.createElement('div');
    seller.className = 'meta';
    // 不寫產地（礦區直送／匿名賣家都不揭露），只留檔位
    seller.textContent = ['公斤料', '表現料', '開窗料'][l.grade];
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
      <button class="btn">${it.price ? fmt(it.price) + ' 喵喵幣' : '按石頭計價'}</button>`;
    card.querySelector('button').addEventListener('click', async () => {
      try {
        const res = await api('POST', '/api/exchange/buy', { key: it.key });
        if (res.payouts) toast(`刮到爽！十顆共 +${fmt(res.total)} 喵喵幣`);
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
  try { const r = await api('POST', '/api/relief', { option: 'ticket' }); toast(`刮到爽 +${fmt(r.total)} 喵喵幣`); refreshMe(); }
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
  table.innerHTML = '<tr><th>#</th><th>玩家</th><th>喵喵幣</th></tr>' +
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
  $('#rank-table').innerHTML = '<tr><th>#</th><th>玩家</th><th>喵喵幣</th></tr>' +
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
      <b>磨石</b>——六層階梯 ×1.3 → ×7.5，每層稀有度更高、成功機率更低，磨崩救回三成，隨時落袋。</p>
    <div class="row" style="margin:10px 0">
      <input id="yb-stake" type="number" min="100" max="1000000" step="100" value="1000"
        style="flex:1;background:var(--panel2);border:1px solid var(--line);color:var(--text);padding:10px;border-radius:8px">
    </div>
    <div class="row">
      <button class="btn" id="yb-cut">切一刀</button>
      <button class="btn danger" id="yb-polish">開始磨石</button>
    </div>
    <div id="yb-odds" style="margin:10px 0"></div>
    <div id="yb-stage" style="margin-top:12px"></div>
    <p style="font-size:12px;color:var(--muted);margin-top:8px" id="yb-err"></p>`;
  bg.appendChild(m);
  document.body.appendChild(bg);
  bg.addEventListener('click', (ev) => { if (ev.target === bg) bg.remove(); });
  const err = m.querySelector('#yb-err');
  const stage = m.querySelector('#yb-stage');
  const oddsBox = m.querySelector('#yb-odds');
  // 賠率表直接由後端來（前後端不會各寫一份）
  api('GET', '/api/yboss/odds').then((o) => {
    oddsBox.innerHTML = '<div style="display:flex;flex-wrap:wrap;gap:10px;font-size:12px;color:var(--muted)">'
      + (o.cut || []).map((e) => `<div style="flex:1 1 60px;border-left:2px solid var(--line);padding-left:6px">
          <div style="color:var(--text)">${esc(e.label)}</div>
          <div style="color:var(--gold)">×${e.mult}</div>
          <div>${Math.round(e.prob * 100)}%</div></div>`).join('')
      + '</div>';
  }).catch(() => {});

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
        if (ybcv && ['磚頭料','豆種','油青種','糯種','冰種','玻璃種'].includes(res.label)) {
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

// ---------- 轉賬 ----------
// 不填條件 = 即時到賬；填了條件先扣錢託管，對方接受才入賬（拒絕/取消全額退還）。
async function loadTransfers() {
  const d = await api('GET', '/api/transfers');
  const box = (id, rows, render, empty) => {
    const el = $(id);
    if (!el) return;
    el.innerHTML = rows.length ? rows.map(render).join('')
      : `<div style="font-size:13px;color:var(--muted)">${empty}</div>`;
  };
  box('#tf-incoming', d.incoming || [], (r) => `
    <div class="card" style="background:rgba(255,255,255,.03);margin-bottom:8px">
      <div><b>${esc(r.from)}</b> 要轉 <b style="color:var(--gold)">${fmt(r.amount)}</b> 喵喵幣給你</div>
      ${r.condition ? `<div style="font-size:13px;color:var(--muted);margin:4px 0">條件：${esc(r.condition)}</div>` : ''}
      <div class="row" style="margin-top:6px">
        <button class="btn" data-tf-acc="${r.id}">接受</button>
        <button class="btn ghost" data-tf-dec="${r.id}">拒絕</button>
      </div>
    </div>`, '目前沒人轉喵喵幣給你。');
  box('#tf-outgoing', d.outgoing || [], (r) => `
    <div class="card" style="background:rgba(255,255,255,.03);margin-bottom:8px">
      <div>等 <b>${esc(r.to)}</b> 接受：<b style="color:var(--gold)">${fmt(r.amount)}</b> 喵喵幣</div>
      ${r.condition ? `<div style="font-size:13px;color:var(--muted);margin:4px 0">條件：${esc(r.condition)}</div>` : ''}
      <div class="row" style="margin-top:6px"><button class="btn ghost" data-tf-can="${r.id}">取消並取回</button></div>
    </div>`, '沒有等待中的轉賬。');
  box('#tf-history', d.history || [], (r) => {
    const st = { sent: '已到賬', accepted: '已接受', declined: '被拒絕', cancelled: '已取消' }[r.status] || r.status;
    return `<div style="font-size:13px;padding:4px 0;border-bottom:1px solid var(--border)">
      ${esc(r.from)} → ${esc(r.to)} · ${fmt(r.amount)} 喵喵幣 · <span style="color:var(--muted)">${st}</span>
      ${r.condition ? `<span style="color:var(--muted)"> · 條件：${esc(r.condition)}</span>` : ''}</div>`;
  }, '還沒有紀錄。');

  document.querySelectorAll('[data-tf-acc]').forEach((b) => b.onclick = () => tfAct('accept', +b.dataset.tfAcc));
  document.querySelectorAll('[data-tf-dec]').forEach((b) => b.onclick = () => tfAct('decline', +b.dataset.tfDec));
  document.querySelectorAll('[data-tf-can]').forEach((b) => b.onclick = () => tfAct('cancel', +b.dataset.tfCan));
}

async function tfAct(kind, id) {
  try {
    const r = await api('POST', '/api/transfer/' + kind, { id });
    if (r.chips !== undefined) setChips(r.chips);
    toast(r.message || '完成', true);
    await loadTransfers();
  } catch (e) { toast(e.message); }
}

const tfSend = document.getElementById('tf-send');
if (tfSend) tfSend.onclick = async () => {
  const to = $('#tf-to').value.trim();
  const amount = parseInt($('#tf-amount').value, 10);
  const condition = $('#tf-cond').value.trim();
  if (!to) return toast('請填對方名稱');
  if (!(amount > 0)) return toast('金額要大於 0');
  if (condition) {
    const ok = confirm(`附帶條件：\n「${condition}」\n\n先扣 ${fmt(amount)} 喵喵幣託管，等 ${to} 接受才成交。確定送出？`);
    if (!ok) return;
  }
  try {
    const r = await api('POST', '/api/transfer', { to, amount, condition });
    if (r.chips !== undefined) setChips(r.chips);
    $('#tf-to').value = ''; $('#tf-amount').value = ''; $('#tf-cond').value = '';
    toast(r.message || '已送出', true);
    await loadTransfers();
  } catch (e) { toast(e.message); }
};


// ---------- 活動公告 ----------
async function loadEvents() {
  const box = $('#event-banner');
  if (!box) return;
  try {
    const d = await api('GET', '/api/events');
    const evs = d.events || [];
    box.innerHTML = evs.map((e) => `
      <div class="card" style="margin-bottom:12px;border-color:var(--gold)">
        <b style="color:var(--gold)">📣 ${esc(e.title)}</b>
        <span style="font-size:12px;color:var(--muted)"> · 剩 ${e.hours_left} 小時</span>
        ${e.body ? `<div style="font-size:13px;margin-top:6px;white-space:pre-wrap">${esc(e.body)}</div>` : ''}
      </div>`).join('');
  } catch { box.innerHTML = ''; }
}

// ---------- 管理員控制臺 ----------
let HIDE_MOCK = localStorage.getItem('hide_mock') === '1';
// 安全寫入：元素不存在也不要讓整段渲染炸掉（列表消失的真兇）
const putHTML = (sel, html) => { const el = $(sel); if (el) el.innerHTML = html; else console.warn('缺元素', sel); };
const putText = (sel, txt) => { const el = $(sel); if (el) el.textContent = txt; else console.warn('缺元素', sel); };
async function loadAdmin() {
  let d = {};
  try {
    d = await api('GET', '/api/admin/panel');
  } catch (e) {
    putHTML('#ad-players', `<div style="color:var(--muted)">控制臺載入失敗：${esc(String(e))}</div>`);
    return;
  }
  const s = d.stats || {};
  if (d.error) {
    putHTML('#ad-players', `<div style="color:var(--muted)">控制臺載入失敗：${esc(d.error)}</div>`);
    return;
  }
  putHTML('#admin-stats', [
    ['玩家總數', s.players], ['真人玩家', s.real_players], ['喵喵幣總量', fmt(s.total_chips)],
    ['倉庫石頭', s.stones_owned], ['已切過', s.stones_cut], ['市場掛單', s.listings_open],
    ['bot 收料（累計／24h）', `${s.bot_buys} / ${s.bot_buys_24h}`],
    ['磨石進行中', s.polish_running], ['待接受轉賬', s.transfers_open],
  ].map(([k, v]) => `<div>${k}：<b style="color:var(--text)">${v}</b></div>`).join(''));

  if (typeof loadAdminRewards === 'function') loadAdminRewards();
  const all = d.players || [];
  const players = all.filter((p) => !HIDE_MOCK || !p.is_mock);
  const total = (d.stats || {}).players || all.length;
  const mockCount = all.filter((p) => p.is_mock).length;
  putText('#ad-players-title',
    `玩家（全部 ${total} 個帳號${HIDE_MOCK ? `，已隱藏 ${mockCount} 個測試帳號` : ''}，依喵喵幣排序）`);
  putHTML('#ad-players', players.length ? players.map((p) => `
    <div style="display:flex;gap:8px;align-items:center;padding:3px 0;border-bottom:1px solid var(--border)">
      <span style="flex:1">${p.is_admin ? '👑 ' : ''}${esc(p.name)}${p.is_mock ? ' <span style="color:var(--muted)">(測試)</span>' : ''}</span>
      <span style="width:90px;text-align:right;color:var(--gold)">${fmt(p.chips)}</span>
      <span style="width:60px;text-align:right;color:var(--muted)">${p.stones} 石</span>
    </div>`).join('') : `<div style="color:var(--muted)">沒有玩家（全部帳號都被隱藏？按「隱藏測試帳號」切換看看）</div>`);

  $('#ad-events').innerHTML = (d.events || []).map((e) => `
    <div style="display:flex;gap:8px;align-items:center;padding:4px 0">
      <span style="flex:1"><b>${esc(e.title)}</b><span style="color:var(--muted);font-size:12px"> · 剩 ${e.hours_left}h</span></span>
      <button class="btn ghost" data-ev-del="${e.id}">下架</button>
    </div>`).join('') || '<div style="font-size:13px;color:var(--muted)">目前沒有活動</div>';

  $('#ad-bots').innerHTML = (d.bots || []).map((b) =>
    `<div>${esc(b.name)}：眼力 <b>${b.eye}×</b>真值 · 放 ${b.sticky_mins >= 60 ? (b.sticky_mins / 60) + ' 小時' : (b.sticky_mins ?? b.sticky_hours * 60 ?? '?') + ' 分鐘'} 後出手 · 每次最多 ${b.appetite} 件</div>`).join('');

  $('#ad-mydiscord').textContent =
    `你的 Discord ID：${d.my_discord_id || '（未取得）'}　—　填進 .env 的 ADMIN_DISCORD_IDS 可以固定管理員身分`;

  document.querySelectorAll('[data-ev-del]').forEach((b) => b.onclick = async () => {
    try { await api('POST', '/api/admin/event/delete', { id: +b.dataset.evDel }); toast('已下架', true); loadAdmin(); loadEvents(); }
    catch (e) { toast(e.message); }
  });
}

async function adminAct(path, body, okMsg) {
  try {
    const r = await api('POST', path, body);
    toast(r.message || okMsg, true);
    if (r.chips !== undefined && me) setChips(me.chips);
    loadAdmin(); loadEvents(); refreshMe();
  } catch (e) { toast(e.message); }
}

const adGrant = document.getElementById('ad-grant');
if (adGrant) adGrant.onclick = () => {
  const user = $('#ad-user').value.trim();
  const amount = parseInt($('#ad-amount').value, 10);
  if (!user || !amount) return toast('要填玩家名稱和金額');
  const verb = amount > 0 ? '發' : '收回';
  if (!confirm(`${verb} ${fmt(Math.abs(amount))} 喵喵幣 ${amount > 0 ? '給' : '從'} ${user}？`)) return;
  adminAct('/api/admin/grant', { user, amount, reason: $('#ad-reason').value.trim() }, '已調整');
};
const adAll = document.getElementById('ad-giveall');
if (adAll) adAll.onclick = () => {
  const amount = parseInt($('#ad-all').value, 10);
  if (!(amount > 0)) return toast('紅包金額要大於 0');
  if (!confirm(`全服每人發 ${fmt(amount)} 喵喵幣？`)) return;
  adminAct('/api/admin/giveall', { amount, reason: '全服紅包' }, '已發紅包');
};
const adPub = document.getElementById('ad-ev-pub');
if (adPub) adPub.onclick = () => {
  const title = $('#ad-ev-title').value.trim();
  if (!title) return toast('活動要有標題');
  adminAct('/api/admin/event', {
    title,
    body: $('#ad-ev-body').value.trim(),
    hours: parseInt($('#ad-ev-hours').value, 10) || 24,
    amount: parseInt($('#ad-ev-amount').value, 10) || 0,
  }, '已發佈').then(() => {
    $('#ad-ev-title').value = ''; $('#ad-ev-body').value = '';
    $('#ad-ev-hours').value = ''; $('#ad-ev-amount').value = '';
  });
};
const adHide = document.getElementById('ad-hide-mock');
if (adHide) {
  const paint = () => { adHide.textContent = HIDE_MOCK ? '顯示測試帳號' : '隱藏測試帳號'; };
  paint();
  adHide.onclick = () => {
    HIDE_MOCK = !HIDE_MOCK;
    localStorage.setItem('hide_mock', HIDE_MOCK ? '1' : '0');
    paint();
    loadAdmin();
  };
}

const adCleanup = document.getElementById('ad-cleanup');
if (adCleanup) adCleanup.onclick = () => {
  if (!confirm('清掉所有 mock: 測試帳號（他們手上的石頭也會消失）？')) return;
  adminAct('/api/admin/cleanup', {}, '已清理');
};


// ---------- 我的紀錄 ----------
async function loadHistory() {
  loadTitles().catch(() => {});
  const d = await api('GET', '/api/history');
  const s = d.stats || {};
  const net = s.net || 0;
  $('#hist-stats').innerHTML = [
    ['處理過的總數', s.total],
    ['切石次數', s.cuts],
    ['勝率（拿回本錢以上）', (s.win_rate || 0) + '%'],
    ['總花費', fmt(s.spent)],
    ['總回收', fmt(s.earned)],
    ['淨賺賠', `<b style="color:${net >= 0 ? 'var(--green)' : 'var(--red)'}">${net >= 0 ? '+' : ''}${fmt(net)}</b>`],
    ['最佳倍率', ((s.best_mult || 0) / 100).toFixed(2) + '×'],
  ].map(([k, v]) => `<div>${k}：<b style="color:var(--text)">${v}</b></div>`).join('');

  const ACT = { cut: '切開', scratch: '刮開', polish: '磨到落袋', sold: '賣掉' };
  const list = $('#hist-list');
  if (!(d.entries || []).length) {
    list.innerHTML = '<div style="color:var(--muted)">還沒有紀錄——去切一顆石頭吧。</div>';
    return;
  }
  list.innerHTML = d.entries.map((e) => {
    const mult = e.price > 0 ? e.payout / e.price : 0;
    const good = e.payout >= e.price;
    return `<div style="display:flex;gap:8px;padding:5px 0;border-bottom:1px solid var(--border)">
      <span style="flex:0 0 78px;color:var(--muted)">${e.created_at.slice(5, 16)}</span>
      <span style="flex:0 0 62px">${ACT[e.action] || e.action}</span>
      <span style="flex:1">${esc(e.quality)}${e.variety ? ' · ' + esc(e.variety) : ''}
        <span style="color:var(--muted);font-size:12px">（${esc(e.stone_id.slice(0, 8))}）</span></span>
      <span style="flex:0 0 150px;text-align:right">${fmt(e.price)} → <b style="color:${good ? 'var(--green)' : 'var(--red)'}">${fmt(e.payout)}</b>
        <span style="color:var(--muted)">${mult.toFixed(2)}×</span></span>
    </div>`;
  }).join('');
}


// ---------- 稱號 ----------
const TITLE_RARE = { 1: 'var(--muted)', 2: 'var(--text)', 3: 'var(--gold)', 4: '#e8b3ff' };

async function loadTitles() {
  const box = $('#title-grid');
  if (!box) return;
  const d = await api('GET', '/api/titles');
  box.innerHTML = (d.titles || []).map((t) => {
    const col = TITLE_RARE[t.rare] || 'var(--text)';
    const border = t.equipped ? 'var(--gold)' : 'var(--border)';
    return `<div class="card" style="background:rgba(255,255,255,.03);border-color:${border};padding:10px">
      <div style="color:${col};font-weight:700">${t.unlocked ? '' : '🔒 '}${esc(t.name)}</div>
      <div style="font-size:12px;color:var(--muted);margin:4px 0 8px">${esc(t.desc)}</div>
      ${t.unlocked
        ? (t.equipped
          ? '<button class="btn ghost" data-title-off="1" style="width:100%">卸下</button>'
          : `<button class="btn" data-title="${esc(t.key)}" style="width:100%">裝上</button>`)
        : '<button class="btn ghost" disabled style="width:100%">未解鎖</button>'}
    </div>`;
  }).join('');
  document.querySelectorAll('[data-title]').forEach((b) => b.onclick = async () => {
    try { const r = await api('POST', '/api/titles/equip', { key: b.dataset.title }); toast(r.message, true); await refreshMe(); loadTitles(); }
    catch (e) { toast(e.message); }
  });
  const off = document.querySelector('[data-title-off]');
  if (off) off.onclick = async () => {
    try { await api('POST', '/api/titles/equip', { key: '' }); toast('已卸下稱號'); await refreshMe(); loadTitles(); }
    catch (e) { toast(e.message); }
  };
}


// ---------- 喵喵錢莊 ----------
function bankRow(k, v) {
  return `<tr><td style="padding:3px 0;color:var(--muted);width:130px">${k}</td><td>${v}</td></tr>`;
}

async function loadBank() {
  const d = await api('GET', '/api/bank');
  const t = d.terms || {};
  $('#bank-terms').innerHTML = [
    bankRow('借貸範圍', `${fmt(t.min)} ~ ${fmt(t.max)} 喵喵幣`),
    bankRow('還款期限', `${t.min_hours} ~ ${t.max_hours} 小時（現實時間，最多一天）`),
    bankRow('逾期', `沒收 <b style="color:var(--red)">一半財產</b>`),
    bankRow('申訴', `${t.max_appeal} 輪（被拒絕才可以申訴）`),
    bankRow('AI', t.ai ? '已接上（貓娘在線）' : '未設定金鑰，暫用基本審核'),
  ].join('');

  // 提議：等玩家按「接受」或「拒絕」
  const off = d.offer;
  const offerEl = document.getElementById('bank-offer');
  // 沒提議／沒東西可申訴就把整個卡片收起來（不要留空殼）
  if (offerEl) offerEl.style.display = off ? '' : 'none';
  const appealCard = document.getElementById('bank-appeal-card');
  const canAppeal = !!(off || (d.loan === null && (d.history || []).some((h) => h.status === 'denied')));
  if (appealCard) appealCard.style.display = (off || d.loan) ? '' : (canAppeal ? '' : 'none');
  if (offerEl) {
    offerEl.innerHTML = off ? `
      <h3 style="color:var(--gold);margin-bottom:8px">老闆娘的提議</h3>
      <table style="width:100%;font-size:13px">
        ${bankRow('金額', fmt(off.principal) + ' 喵喵幣')}
        ${bankRow('利率', ((off.rate || 0.25) * 100).toFixed(0) + '%')}
        ${bankRow('到期日（現實時間）', `${off.hours} 小時後`)}
        ${bankRow('到期要還', `<b style="color:var(--gold)">${fmt(off.principal + off.interest)}</b>`)}
      </table>
      <div class="row" style="margin-top:10px">
        <button class="btn" id="bk-accept">接受</button>
        <button class="btn ghost" id="bk-reject">拒絕</button>
      </div>` : '';
    const acc = document.getElementById('bk-accept');
    if (acc) acc.onclick = async () => {
      try { const r = await api('POST', '/api/bank/accept', {}); toast(r.message, true); setChips(r.chips); loadBank(); }
      catch (e) { toast(e.message); }
    };
    const rej = document.getElementById('bk-reject');
    if (rej) rej.onclick = async () => {
      try { const r = await api('POST', '/api/bank/reject', {}); toast(r.message); loadBank(); }
      catch (e) { toast(e.message); }
    };
  }

  const l = d.loan;
  $('#bank-current').innerHTML = l
    ? `<h3 style="color:var(--gold);margin-bottom:8px">目前貸款</h3>
       <table style="width:100%;font-size:13px">
         ${bankRow('本金', fmt(l.principal))}
         ${bankRow('利息', `${fmt(l.interest)}（${(l.rate ? l.rate * 100 : 25).toFixed(0)}%）`)}
         ${l.penalty ? bankRow('違約金', `<b style="color:var(--red)">${fmt(l.penalty)}</b>`) : ''}
         ${bankRow('到期要還', `<b style="color:var(--gold)">${fmt(d.repay_total)}</b>`)}
         ${bankRow('到期時間（UTC）', l.due_at)}
         ${bankRow('已申訴', `${l.appeals} / ${t.max_appeal} 輪`)}
       </table>
       <div class="row" style="margin-top:10px"><button class="btn" id="bk-repay">還清 ${fmt(d.repay_total)}</button></div>`
    : '<div style="font-size:13px;color:var(--muted)">目前沒有欠錢喵。</div>';
  const rp = document.getElementById('bk-repay');
  if (rp) rp.onclick = async () => {
    if (!confirm('確定還清？')) return;
    try { const r = await api('POST', '/api/bank/repay', {}); toast(r.message, true); setChips(r.chips); loadBank(); }
    catch (e) { toast(e.message); }
  };

  const chat = (d.chat || []);
  // 對話框形式：老闆娘在左（貓娘頭像），玩家在右
  const bubble = (role, text) => role === 'banker'
    ? `<div class="chat-row them"><div class="chat-av">🐱</div>
         <div><div class="chat-name">喵喵老闆娘</div><div class="chat-bubble">${esc(text)}</div></div></div>`
    : `<div class="chat-row me"><div class="chat-av">🙋</div>
         <div><div class="chat-name" style="text-align:right">你</div><div class="chat-bubble">${esc(text)}</div></div></div>`;
  $('#bk-chat').innerHTML = chat.length
    ? chat.map((m) => bubble(m.role, m.content)).join('')
    : bubble('banker', d.greeting || '歡迎光臨喵喵錢莊喵～');

  $('#bank-history').innerHTML = (d.history || []).length
    ? '<tr><th align="left">本金</th><th align="left">利息</th><th align="left">時數</th><th align="left">狀態</th></tr>'
      + d.history.map((h) => `<tr><td>${fmt(h.principal)}</td><td>${fmt(h.interest)}</td><td>${h.hours}h</td>
          <td>${({ repaid: '已還清', active: '進行中', defaulted: '逾期沒收' })[h.status] || h.status}</td></tr>`).join('')
    : '<tr><td style="color:var(--muted)">還沒有紀錄</td></tr>';

  const est = () => {
    const amt = parseInt($('#bk-amount').value, 10) || 0;
    const hrs = parseInt($('#bk-hours').value, 10) || 0;
    if (!amt) { $('#bk-est').textContent = ''; return; }
    const rate = Math.min(Math.max((parseFloat($('#bk-rate')?.value) || 30) / 100, 0.05), 0.60);
    $('#bk-est').textContent = `到期要還 ${fmt(Math.round(amt * (1 + rate)))}`;
  };
  ['#bk-amount', '#bk-hours', '#bk-rate'].forEach((s) => { const el = $(s); if (el) el.oninput = est; });
  est();
  const ap = document.getElementById('bk-apply');
  if (ap) ap.onclick = async () => {
    try {
      const r = await api('POST', '/api/bank/apply', {
        amount: parseInt($('#bk-amount').value, 10) || 0,
        hours: parseInt($('#bk-hours').value, 10) || 0,
        reason: $('#bk-reason').value.trim(),
        rate: (parseFloat($('#bk-rate')?.value) || 30) / 100,
      });
      toast(r.message, r.decision !== 'deny');
      if (r.chips !== undefined) setChips(r.chips);
      loadBank();
    } catch (e) { toast(e.message); }
  };
  const sd = document.getElementById('bk-send');
  if (sd) sd.onclick = async () => {
    const msg = $('#bk-say').value.trim();
    if (!msg) return;
    {
      const chat = $('#bk-chat');
      const say = (role, text) => {
        if (chat) { chat.insertAdjacentHTML('beforeend', bubble(role, text)); chat.scrollTop = chat.scrollHeight; }
      };
      $('#bk-say').value = '';
      say('me', msg); // 先把自己說的話顯示出來（就算申訴被擋，也不能讓玩家覺得訊息消失了）
      try {
        const r = await api('POST', '/api/bank/appeal', { message: msg });
        if (r && r.message) say('them', r.message);
        toast(r.message, true);
        loadBank();
      } catch (e) {
        say('them', e.message || String(e));
        toast(e.message || String(e));
      }
    }
  };
}

// 寶石圖鑒（2026-09-17 用戶要求）：切到過的彩蛋寶石永久收集
async function loadGemBook() {
  const host = document.querySelector('#view-collection');
  if (!host) return;
  let box = document.getElementById('gem-book');
  if (!box) {
    box = document.createElement('div');
    box.id = 'gem-book';
    box.style.marginTop = '16px';
    host.appendChild(box);
  }
  let d;
  try { d = await api('GET', '/api/gems'); }
  catch (e) {
    box.innerHTML = '<div class="card"><h3 style="color:var(--gold)">💎 寶石圖鑒</h3>' +
      '<div style="color:var(--muted);font-size:13px">讀取失敗：' + esc(String(e)) + '</div></div>';
    return;
  }
  const list = d.gems || [];
  const cards = list.map((g, i) => {
    const got = (g.count || 0) > 0;
    const pct = ((g.prob || 0) * (d.chance || 0) * 100).toFixed(2);
    return '<div class="card" style="text-align:center;padding:10px;' + (got ? '' : 'opacity:.55;') + '">' +
      (got ? '<canvas id="gem-cv-' + i + '" width="120" height="120" style="width:104px;height:104px"></canvas>'
           : '<div style="font-size:38px;padding:16px 0">❔</div>') +
      '<div style="font-weight:700;margin-top:4px;font-size:14px">' + (got ? esc(g.name) : '？？？') + '</div>' +
      '<div style="font-size:12px;color:var(--muted)">' +
        (got ? ('切到 ' + g.count + ' 次 · 首次 ' + esc(String(g.first_at || '').slice(0, 10)))
             : ('未收集 · 整體 ' + pct + '%')) + '</div>' +
      '<div style="font-size:12px;color:var(--gold)">價值 ×' + Number(g.mult).toFixed(2) + '</div>' +
      '</div>';
  }).join('');
  box.innerHTML = '<div class="card"><h3 style="color:var(--gold);margin-bottom:4px">💎 寶石圖鑒</h3>' +
    '<div style="font-size:12px;color:var(--muted);margin-bottom:10px">切石時 5% 會切出「不是玉石」的寶石彩蛋；收集 ' +
    (d.kinds_owned || 0) + ' / ' + (d.kinds_total || 0) + ' 種，共 ' + (d.count_total || 0) + ' 顆</div>' +
    '<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(118px,1fr));gap:10px">' + cards + '</div></div>';
  list.forEach((g, i) => {
    if ((g.count || 0) > 0 && window.StoneRender && StoneRender.cutView) {
      const cv = document.getElementById('gem-cv-' + i);
      if (cv) { try { StoneRender.cutView(cv, 1234 + i * 77, g.key, 'base', { grade: 0 }); } catch (e) {} }
    }
  });
}
