// 只要奪寶頁是開著的就不停更新（大廳排隊也要即時，不能只有坐在桌上才更新）
function heistEnsurePoll() {
  if (HEIST_POLL) return;
  HEIST_POLL = setInterval(() => {
    const v = document.getElementById('view-heist');
    if (v && v.classList.contains('active')) { loadHeist(); heistSyncChips(); }
  }, 2500);
}

// 上一輪投背叛打我的人是誰（只有死裏逃生那位看得到，名字由後端 hunters 提供）
function heistHunters(h) {
  if (!h || !Array.isArray(h.hunters)) return [];
  return h.hunters.filter(Boolean);
}


// ---------- 奪寶（四人囚徒困境）UI ----------
let HEIST_POLL = null;
const HEIST_GRADES = [['kilo', '公斤料', 0], ['feature', '表現料', 1], ['window', '開窗料', 2]];

function heistStopPoll() { if (HEIST_POLL) { clearInterval(HEIST_POLL); HEIST_POLL = null; } }

async function loadHeist() {
  heistEnsurePoll();
  const el = document.getElementById('heist-body');
  if (!el) return;
  let d;
  try {
    d = await api('GET', '/api/heist');
  } catch (e) {
    {const __sy=window.scrollY; el.innerHTML = `<div class="card">載入失敗：${esc(String(e))}</div>`; window.scrollTo(0, __sy);}
    return;
  }
  renderHeist(d);
  try { renderHeistChat(d); } catch (e) {}
}

function heistBar(p, t) {
  const pct = t > 0 ? Math.min(100, Math.round((p / t) * 100)) : 0;
  return `<div style="background:var(--panel2);border:1px solid var(--line);border-radius:8px;height:22px;overflow:hidden;position:relative">
    <div style="width:${pct}%;height:100%;background:linear-gradient(90deg,#7b5cff,#c8a15a)"></div>
    <div style="position:absolute;inset:0;text-align:center;font-size:13px;line-height:22px">進度 ${p} / ${t}　（${pct}%）</div>
  </div>`;
}

function renderHeist(d) {
  const el = document.getElementById('heist-body');
  const h = d.heist;
  if (!h) {
    heistStopPoll();
    const rules = esc(d.rules || '');
    el.innerHTML = `
      <div class="card"><h3>奪寶</h3>
        <div style="font-size:13px;color:var(--muted);line-height:1.7;margin-bottom:10px">${rules}</div>
        <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(190px,1fr));gap:10px">
          ${HEIST_GRADES.map(([k, name, g]) => `
            <div style="background:var(--panel2);border:1px solid var(--line);border-radius:10px;padding:12px">
              <div style="font-weight:700;margin-bottom:6px">${name}</div>
              <div style="font-size:13px;color:var(--muted)">入場費 <b style="color:var(--text)">${fmt(d.fees[k])}</b></div>
              <div style="font-size:13px;color:var(--muted)">寶石價值 <b style="color:var(--gold)">${fmt(d.pots[k])}</b></div>
              <div style="font-size:12px;color:var(--muted);margin:6px 0">挖到才發獎｜四人平分各 ${fmt(Math.floor(d.pots[k] / 4))}</div>
              <button type="button" onclick="heistJoin(${g})" style="width:100%">入場</button>
            </div>`).join('')}
        </div>
        <div style="font-size:12px;color:var(--muted);margin-top:10px">沒真人時 5 分鐘才會自動補 bot（bot 各有脾氣：記仇的、貪財的、佛系的…）；不想等就按桌上的「直接開局」。</div>
      </div>
      <div class="card"><h3>排隊中</h3>
        ${(d.queues || []).length ? (d.queues || []).map((q) => {
          const name = (HEIST_GRADES[q.grade] || [null, '未知'])[1];
          return `<div style="padding:6px 0;border-bottom:1px solid var(--border);font-size:13px">
            <b>${name}桌 #${q.id}</b>　${q.count}/${q.need} 人
            ${q.missing > 0 ? `<span style="color:var(--gold)">還缺 ${q.missing} 人成團</span>` : '<span style="color:#5fbf7f">已開局</span>'}
            <div style="color:var(--muted);font-size:12px">${(q.names || []).map(esc).join('、')}${q.bot_in > 0 && q.missing > 0 ? `　（${heistMmss(q.bot_in)} 後補 bot）` : ''}</div>
          </div>`;
        }).join('') : '<div style="color:var(--muted);font-size:13px">目前沒有人在排隊，你可以先開一桌</div>'}
      </div>`;
    return;
  }
  const me = d.me || {};
  const others = d.others || [];

// 凶手名字查表（me + others 都可能是凶手）
function heistKillerName(killerID, me, others) {
  if (me.user_id === killerID) return me.name;
  const o = (others || []).find(x => x.user_id === killerID);
  return o ? o.name : '';
}
  const done = h.status === 'done';
  const alive = others.filter((o) => o.alive).length + (me.alive ? 1 : 0);
  const log = (d.history || []).slice(0, 5);

  let endText = '';
  if (done) {
    if (me.payout > 0) endText = `🎉 這局結束了：你分到 <b style="color:var(--gold)">${fmt(me.payout)}</b>`;
    else if (!me.alive) endText = `💀 這局結束了：你死在這趟奪寶裡`;
    else endText = `這局結束了：什麼都沒拿到（崩塌／沒挖到）`;
  }

  el.innerHTML = `
    <div class="card">
      <h3>奪寶桌 #${h.id}${heistCd(h)}　第 ${h.round}/${d.rounds} 輪</h3>
      <div style="font-size:13px;color:var(--muted);margin-bottom:8px">
        狀態：${done ? '已結束' : h.status === 'running' ? '進行中' : '等人（還缺 ' + (h.need - h.seats) + ' 人，5 分鐘後自動補 bot）'}
        　·　入場費 ${fmt(h.entry)}　·　寶石價值 <b style="color:var(--gold)">${fmt(h.pot)}</b>　·　活著 ${alive} 人
      </div>
      ${heistBar(h.progress, h.target)}
      ${endText ? `<div style="margin-top:10px;padding:10px;border-radius:8px;background:var(--panel2);border:1px solid var(--line)">${endText}</div>` : ''}
      <div style="margin-top:12px;display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:8px">
        ${[{ user_id: me.user_id, name: me.name, alive: me.alive, payout: me.payout, tried: me.tried_to_kill_me, killed_by: me.killed_by, mine: true }, ...others.map(o => ({ ...o, mine: false }))]
      .map((s) => `
          <div class="seat" style="padding:8px;border-radius:8px;border:1px solid ${s.mine ? 'var(--gold)' : 'var(--line)'};background:var(--panel2);${s.alive ? '' : 'opacity:.5'}">
            <div style="font-size:13px">${s.alive ? '🪨' : '☠️'} ${esc(s.name)}${s.mine ? '（你）' : ''}</div>
            ${s.payout > 0 ? `<div style="font-size:12px;color:var(--gold)">分到 ${fmt(s.payout)}</div>` : ''}
            ${s.killed_by ? (() => { const kb = Math.abs(s.killed_by); const nm = heistKillerName(kb, me, others);
              return s.killed_by < 0
                ? `<div style="font-size:12px;color:#e09c6c">🛡 反殺了 ${esc(nm || '?')}（他先出手殺你，被你做掉）</div>`
                : `<div style="font-size:12px;color:#e06c6c">🔪 被殺了${nm ? '（凶手：' + esc(nm) + '）' : ''}</div>`;
            })() : ''}
            ${s.tried ? `<div style="font-size:12px;color:#e06c6c">⚠ ${heistKillerName(s.tried, me, others) || heistHunters(h).join('、') || '有人'} 想殺你（已曝光）</div>` : ''}
            ${!s.mine && s.alive && !done ? `<div style="margin-top:4px"><button type="button" onclick="heistAct('betray',${s.user_id})" class="hbtn hbtn-betray hbtn-sm">🔪 背叛他</button></div>` : ''}
          </div>`).join('')}
      </div>
      ${(done || h.status === 'open') ? `<div style="margin-top:12px;display:flex;gap:8px;flex-wrap:wrap">
        ${h.status === 'open' ? `<button type="button" onclick="heistFill()" class="hbtn hbtn-pill">🎲 直接開局（補 bot）</button>` : ''}
        <button type="button" onclick="heistLeave()" class="hbtn hbtn-ghost hbtn-pill">🚪 ${done ? '離開桌子' : '退出排隊（退還入場費）'}</button>
      </div>` : `
      ${!me.alive ? `<div style="margin-top:12px"><button type="button" onclick="heistLeave()" class="hbtn hbtn-ghost hbtn-pill">🚪 已出局，離開桌子喵</button></div>` : `
      <div class="hrow" style="margin-top:12px;display:flex;gap:8px;flex-wrap:wrap">
        <button type="button" onclick="heistAct('cooperate',0)" class="hbtn hbtn-coop hbtn-pill">🤝 合作（推進度）</button>
        <span style="font-size:12px;color:var(--muted);align-self:center">或按上面某個對手的「背叛他」——一輪只能對一個人下手</span>
      </div>
      <div style="font-size:12px;color:var(--muted);margin-top:8px">${me.my_action === 'betray' ? '你這一輪已出手：<b>🔪 背叛</b>' : me.my_action === 'cooperate' ? '你這一輪已出手：<b>🤝 合作</b>' : '還沒出手（30 秒內出手，全員出手即結算）'}</div>`}`}
    </div>
    ${log.length ? `<div class="card"><h3>我的奪寶紀錄</h3>${log.map((r) => `
      <div style="font-size:13px;padding:3px 0;border-bottom:1px solid var(--border)">
        桌 #${r.heist_id}　進度 ${r.progress}/${r.target}　${r.status === 'done' ? (r.alive ? (r.payout > 0 ? '<span style="color:var(--gold)">分到 ' + fmt(r.payout) + '</span>' : '沒挖到') : '<span style="color:#e06c6c">死亡</span>') : '進行中'}
      </div>`).join('')}</div>` : ''}`;

  if (!done) {
    heistStopPoll();
    heistEnsurePoll();
  } else {
    heistStopPoll();
  }
}

async function heistJoin(grade) {
  try {
    const r = await api('POST', '/api/heist/join', { grade });
    toast(r.message || '已入場');
    await heistSyncChips();
  } catch (e) { toast(String(e)); }
  await heistSyncChips();
  loadHeist();
}

async function heistAct(action, target) {
  try {
    await api('POST', '/api/heist/act', { action, target: target || 0 });
    await heistSyncChips();
  } catch (e) { toast(String(e)); }
  await heistSyncChips();
  loadHeist();
}

async function heistFill() {
  try {
    const r = await api('POST', '/api/heist/fill');
    toast(r.message || '已開局');
  } catch (e) { toast(String(e)); }
  await heistSyncChips();
  loadHeist();
}

async function heistLeave() {
  try {
    await api('POST', '/api/heist/leave');
    toast('已離開');
    await heistSyncChips();
  } catch (e) { toast(String(e)); }
  await heistSyncChips();
  loadHeist();
}

// 奪寶可愛按鈕樣式（喵喵幣風格）
if (!document.getElementById('heist-style')) {
  const st = document.createElement('style');
  st.id = 'heist-style';
  st.textContent = `
  .hbtn-sm{padding:6px 14px;font-size:13px}
  @media (max-width: 680px) {
    .hbtn{width:100%;justify-content:center;display:flex;padding:13px 16px;font-size:15px}
    .hbtn-sm{width:100%;padding:9px 12px;font-size:14px}
    .hrow{flex-direction:column;gap:10px}
    #heist-body > div > div{grid-template-columns:repeat(2,1fr) !important}
    #heist-body .seat{padding:10px 8px}
    #heist-body .card{padding:12px 10px}
  }
  .hbtn{border:0;border-radius:999px;padding:10px 20px;font-size:14px;font-weight:700;color:#3a2a05;
    cursor:pointer;background:linear-gradient(180deg,#ffd977,#f0b93c);
    box-shadow:0 3px 0 #b8862a,0 6px 14px rgba(0,0,0,.35);transition:transform .08s,box-shadow .08s;letter-spacing:.3px}
  .hbtn:hover{transform:translateY(-1px);box-shadow:0 4px 0 #b8862a,0 8px 18px rgba(0,0,0,.4)}
  .hbtn:active{transform:translateY(2px);box-shadow:0 1px 0 #b8862a,0 2px 6px rgba(0,0,0,.35)}
  .hbtn:disabled{opacity:.45;cursor:not-allowed;transform:none}
  .hbtn-coop{background:linear-gradient(180deg,#a8e6a1,#5cbf5c);box-shadow:0 3px 0 #2f7a34,0 6px 14px rgba(0,0,0,.35);color:#0d2f12}
  .hbtn-betray{background:linear-gradient(180deg,#ffb3b3,#e05555);box-shadow:0 3px 0 #8f2b2b,0 6px 14px rgba(0,0,0,.35);color:#3d0b0b}
  .hbtn-ghost{background:linear-gradient(180deg,#5b5b66,#40404a);color:#ffe9b0;box-shadow:0 3px 0 #232329,0 6px 14px rgba(0,0,0,.35)}
  .hbtn-pill{display:inline-flex;align-items:center;gap:6px}
  .heist-coin{width:18px;height:18px;vertical-align:-4px;margin-right:2px}
  `;
  document.head.appendChild(st);
}

// 喵喵幣餘額即時同步（結算後不用重整就該看到錢進來）
async function heistSyncChips() {
  try {
    const me = await api('GET', '/api/me');
    const v = me && (me.chips !== undefined ? me.chips : me.Chips);
    if (typeof v === 'number') {
      const el = document.querySelector('#chips');
      if (el) el.textContent = v.toLocaleString();
    }
  } catch (e) { /* 沒登入就算了 */ }
}

// mm:ss
function heistMmss(sec) {
  sec = Math.max(0, Math.round(sec));
  return Math.floor(sec / 60) + ':' + String(sec % 60).padStart(2, '0');
}

// 行動倒數（一輪 30 秒，沒出手自動算合作）
function heistCd(h) {
  if (!h) return '';
  if (h.status === 'open') {
    const s = Number(h.bot_in !== undefined ? h.bot_in : 300);
    window.__heistDeadline = Date.now() + Math.max(0, s) * 1000;
    window.__heistCdMode = 'fill';
    return '<span id="heist-cd" style="margin-left:10px;font-weight:700;color:#ffd977">\u23f3 自動補 bot ' + heistMmss(s) + '</span>';
  }
  if (h.status !== 'running') return '';
  window.__heistCdMode = 'round';
  const sec = Number(h.round_left !== undefined ? h.round_left : 30);
  window.__heistDeadline = Date.now() + Math.max(0, sec) * 1000;
  return `<span id="heist-cd" style="margin-left:10px;font-weight:700;color:#ffd977">⏳ ${Math.max(0, sec)} 秒</span>`;
}
setInterval(() => {
  const el = document.getElementById('heist-cd');
  if (!el || !window.__heistDeadline) return;
  const left = Math.max(0, Math.round((window.__heistDeadline - Date.now()) / 1000));
  if (window.__heistCdMode === 'fill') {
    el.textContent = left > 0 ? ('⏳ 自動補 bot ' + heistMmss(left)) : '⏳ 正在補 bot…';
  } else {
    el.textContent = left > 0 ? `⏳ ${left} 秒` : '⏳ 沒出手＝自動合作…';
  }
  el.style.color = left <= 5 ? '#ff8a8a' : '#ffd977';
}, 1000);

heistEnsurePoll();

// ---------- 同桌嘴砲＋貓貓表情包（OpenMoji CC BY-SA 4.0）----------
const HEIST_CAT_MEMES = [
  { key: 'grin', file: '1F638.svg', words: '好耶！' },
  { key: 'joy', file: '1F639.svg', words: '笑死喵！' },
  { key: 'love', file: '1F63B.svg', words: '這顆我愛了！' },
  { key: 'smirk', file: '1F63C.svg', words: '這把我全都要' },
  { key: 'kiss', file: '1F63D.svg', words: '合作挖石喵！' },
  { key: 'angry', file: '1F63E.svg', words: '你等著喵！' },
  { key: 'cry', file: '1F63F.svg', words: '不要殺我喵…' },
  { key: 'shock', file: '1F640.svg', words: '完蛋了喵！' },
  { key: 'smile', file: '1F63A.svg', words: '今天心情好喵' },
  { key: 'watch', file: '1F431.svg', words: '我盯著你喵' },
  { key: 'walk', file: '1F408.svg', words: '我先溜了喵' },
  { key: 'black', file: '1F408-200D-2B1B.svg', words: '偷偷摸摸喵' },
  { key: 'tiger', file: '1F42F.svg', words: '兇起來了喵！' },
  { key: 'lion', file: '1F981.svg', words: '王者登場喵！' },
  { key: 'b-angry', file: 'bitty-angry.svg', words: '真的生氣了！' },
  { key: 'b-cool', file: 'bitty-cool.svg', words: '這把穩了' },
  { key: 'b-happy', file: 'bitty-happy.svg', words: '開心到起飛！' },
  { key: 'b-kiss', file: 'bitty-kiss.svg', words: '給你一個親親' },
  { key: 'b-laugh', file: 'bitty-laugh.svg', words: '哈哈哈哈哈！' },
  { key: 'b-sad', file: 'bitty-sad.svg', words: '貓貓難過…' },
  { key: 'b-speechless', file: 'bitty-speechless.svg', words: '我無言了喵' },
  { key: 'b-tongue', file: 'bitty-tongue.svg', words: '略略略～' },
  { key: 'b-wink', file: 'bitty-wink.svg', words: '你懂的喵' },
  { key: 'b-wow', file: 'bitty-wow.svg', words: '真的假的？！' }
];

function heistChatBox() {
  const view = document.getElementById('view-heist');
  if (!view) return null;
  let box = document.getElementById('heist-chat');
  if (!box) {
    box = document.createElement('div');
    box.className = 'card';
    box.id = 'heist-chat';
    box.style.marginTop = '12px';
    const me = (window.__me && window.__me.username) ? window.__me.username : '';
    box.innerHTML = '<div style="font-size:13px;color:var(--gold);font-weight:700;margin-bottom:6px">💬 桌面嘴砲</div>' +
      '<div id="heist-chat-log" style="max-height:150px;overflow-y:auto;display:flex;flex-direction:column;gap:4px;margin-bottom:8px"></div>' +
      '<div id="heist-cat-memes" style="display:flex;gap:5px;overflow-x:auto;padding:2px 0 8px"></div>' +
      '<div class="row" style="gap:6px">' +
      '<input id="heist-chat-in" maxlength="80" placeholder="說點什麼…（2 秒一句）" style="flex:1">' +
      '<button class="btn" id="heist-chat-send">送出</button></div>';
    view.appendChild(box);
    const postChat = async (text) => {
      const clean = (text || '').trim();
      if (!clean) return;
      try {
        const r = await api('POST', '/api/heist/say', { text: clean });
        if (r && r.chat) paintHeistChat(r.chat);
      } catch (e) { toast(e.message || String(e)); }
    };
    const send = async () => {
      const inp = document.getElementById('heist-chat-in');
      const text = (inp.value || '').trim();
      if (!text) return;
      inp.value = '';
      await postChat(text);
    };
    const memeBar = box.querySelector('#heist-cat-memes');
    HEIST_CAT_MEMES.forEach((meme) => {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'btn';
      btn.title = meme.words;
      btn.style.cssText = 'flex:0 0 auto;width:58px;height:54px;padding:3px;line-height:1';
      btn.innerHTML = '<img src="/cat-stickers/' + meme.file + '" alt="' + esc(meme.words) + '" style="width:42px;height:42px;display:block;margin:auto">';
      btn.addEventListener('click', () => postChat('[cat:' + meme.key + '] ' + meme.words));
      memeBar.appendChild(btn);
    });
    box.querySelector('#heist-chat-send').addEventListener('click', send);
    box.querySelector('#heist-chat-in').addEventListener('keydown', (ev) => { if (ev.key === 'Enter') send(); });
  }
  return box;
}

// paintHeistChat: 畫出對話（自己的話套泡泡框收藏）
function paintHeistChat(chat) {
  const log = document.getElementById('heist-chat-log');
  if (!log) return;
  const meName = (window.__me && window.__me.username) ? window.__me.username : '';
  const bubble = (window.__me && window.__me.bubble) ? window.__me.bubble : '';
  log.innerHTML = (chat || []).map((m) => {
    const mine = meName && m.name === meName;
    const cls = mine && bubble ? (' ' + bubble) : '';
    const right = mine ? 'text-align:right' : '';
    const sticker = /^\[cat:([a-z]+)\]\s*(.*)$/.exec(m.text || '');
    const meme = sticker ? HEIST_CAT_MEMES.find((x) => x.key === sticker[1]) : null;
    const body = meme
      ? '<span class="chat-bubble" style="display:inline-flex;align-items:center;gap:7px;padding:5px 9px;border-radius:12px;background:var(--panel2)">' +
        '<img src="/cat-stickers/' + meme.file + '" alt="貓貓表情" style="width:48px;height:48px">' +
        '<span>' + esc(sticker[2] || meme.words) + '</span></span>'
      : '<span class="chat-bubble" style="display:inline-block;padding:3px 9px;border-radius:10px;background:var(--panel2)">' + esc(m.text || '') + '</span>';
    return '<div class="chat-row' + cls + '" style="' + right + '">' +
      '<span style="font-size:12px;color:var(--muted)">' + esc(m.name || '') + '</span> ' +
      body + '</div>';
  }).join('') || '<div style="font-size:12px;color:var(--muted)">還沒有人講話…</div>';
  log.scrollTop = log.scrollHeight;
}

function renderHeistChat(d) {
  heistChatBox();
  paintHeistChat(d && d.chat);
}
