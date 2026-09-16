
// ---------- 奪寶（四人囚徒困境）UI ----------
let HEIST_POLL = null;
const HEIST_GRADES = [['kilo', '公斤料', 0], ['feature', '表現料', 1], ['window', '開窗料', 2]];

function heistStopPoll() { if (HEIST_POLL) { clearInterval(HEIST_POLL); HEIST_POLL = null; } }

async function loadHeist() {
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
              <div style="font-size:12px;color:var(--muted);margin:6px 0">平分每人 ${fmt(Math.floor(d.pots[k] / 4))}｜獨吞 ${fmt(d.pots[k])}</div>
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
            <div style="color:var(--muted);font-size:12px">${(q.names || []).map(esc).join('、')}${q.bot_in > 0 && q.missing > 0 ? `　（${q.bot_in} 秒後補 bot）` : ''}</div>
          </div>`;
        }).join('') : '<div style="color:var(--muted);font-size:13px">目前沒有人在排隊，你可以先開一桌</div>'}
      </div>`;
    return;
  }
  const me = d.me || {};
  const others = d.others || [];
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
        ${[{ user_id: me.user_id, name: me.name, alive: me.alive, payout: me.payout, tried: me.tried_to_kill_me, mine: true }, ...others.map(o => ({ ...o, mine: false }))]
      .map((s) => `
          <div style="padding:8px;border-radius:8px;border:1px solid ${s.mine ? 'var(--gold)' : 'var(--line)'};background:var(--panel2);${s.alive ? '' : 'opacity:.5'}">
            <div style="font-size:13px">${s.alive ? '🪨' : '☠️'} ${esc(s.name)}${s.mine ? '（你）' : ''}</div>
            ${s.payout > 0 ? `<div style="font-size:12px;color:var(--gold)">分到 ${fmt(s.payout)}</div>` : ''}
            ${s.tried ? `<div style="font-size:12px;color:#e06c6c">⚠ 有人想殺你（已曝光）</div>` : ''}
            ${!s.mine && s.alive && !done ? `<div style="margin-top:4px"><button type="button" onclick="heistAct('betray',${s.user_id})" style="font-size:12px;padding:4px 8px">🔪 背叛他</button></div>` : ''}
          </div>`).join('')}
      </div>
      ${(done || h.status === 'open') ? `<div style="margin-top:12px"><button type="button" onclick="heistLeave()" class="hbtn hbtn-ghost hbtn-pill">🚪 ${done ? '離開桌子' : '退出排隊（退還入場費）'}</button></div>` : `
      <div style="margin-top:12px;display:flex;gap:8px;flex-wrap:wrap">
        ${h.status === 'open' ? `<button type="button" onclick="heistFill()" class="hbtn hbtn-pill">🎲 直接開局（補 bot）</button>` : ''}
        <button type="button" onclick="heistAct('cooperate',0)" class="hbtn hbtn-coop hbtn-pill">🤝 合作（推進度）</button>
        <span style="font-size:12px;color:var(--muted);align-self:center">或按上面某個對手的「背叛他」——一輪只能對一個人下手</span>
      </div>
      <div style="font-size:12px;color:var(--muted);margin-top:8px">${me.my_action === 'betray' ? '你這一輪已出手：<b>🔪 背叛</b>' : me.my_action === 'cooperate' ? '你這一輪已出手：<b>🤝 合作</b>' : '還沒出手（30 秒內出手，全員出手即結算）'}</div>`}
    </div>
    ${log.length ? `<div class="card"><h3>我的奪寶紀錄</h3>${log.map((r) => `
      <div style="font-size:13px;padding:3px 0;border-bottom:1px solid var(--border)">
        桌 #${r.heist_id}　進度 ${r.progress}/${r.target}　${r.status === 'done' ? (r.alive ? (r.payout > 0 ? '<span style="color:var(--gold)">分到 ' + fmt(r.payout) + '</span>' : '沒挖到') : '<span style="color:#e06c6c">死亡</span>') : '進行中'}
      </div>`).join('')}</div>` : ''}`;

  if (!done) {
    heistStopPoll();
    HEIST_POLL = setInterval(() => { if (document.getElementById('view-heist').classList.contains('active')) loadHeist(); }, 2500);
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

// 行動倒數（一輪 30 秒，沒出手自動算合作）
function heistCd(h) {
  if (!h || h.status !== 'running') return '';
  const sec = Number(h.round_left !== undefined ? h.round_left : 30);
  window.__heistDeadline = Date.now() + Math.max(0, sec) * 1000;
  return `<span id="heist-cd" style="margin-left:10px;font-weight:700;color:#ffd977">⏳ ${Math.max(0, sec)} 秒</span>`;
}
setInterval(() => {
  const el = document.getElementById('heist-cd');
  if (!el || !window.__heistDeadline) return;
  const left = Math.max(0, Math.round((window.__heistDeadline - Date.now()) / 1000));
  el.textContent = left > 0 ? `⏳ ${left} 秒` : '⏳ 沒出手＝自動合作…';
  el.style.color = left <= 5 ? '#ff8a8a' : '#ffd977';
}, 1000);
