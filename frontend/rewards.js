
// ---------- 獎勵兌換申請（管理員審核）----------
async function loadRewards() {
  const box = document.getElementById('reward-box');
  if (!box) return;
  let d;
  try { d = await api('GET', '/api/rewards'); } catch (e) {
    box.innerHTML = `<div class="card">獎勵載入失敗：${esc(String(e))}</div>`; return;
  }
  const list = d.requests || [];
  const big = (d.presets || [])[0] || { title: '一百萬大獎', cost: 1000000 };
  box.innerHTML = `
    <div class="card">
      <h3>獎勵兌換（站長審核）</h3>
      <div style="font-size:13px;color:var(--muted);margin-bottom:10px">
        送出申請後不會立刻扣喵喵幣——站長在控制臺按下「通過」才會扣，並由站長安排獎勵內容。
      </div>
      <div style="background:var(--panel2);border:1px solid var(--gold);border-radius:10px;padding:12px;margin-bottom:10px">
        <div style="font-weight:700;color:var(--gold)">🏆 ${esc(big.title)}</div>
        <div style="font-size:13px;color:var(--muted);margin:6px 0">需要 ${fmt(big.cost)} 喵喵幣</div>
        <button onclick="reqReward('${esc(big.title)}', ${big.cost}, '')">申請一百萬大獎</button>
      </div>
      <div style="background:var(--panel2);border:1px solid var(--line);border-radius:10px;padding:12px">
        <div style="font-weight:700;margin-bottom:6px">✨ 自訂獎勵</div>
        <div style="display:flex;gap:6px;flex-wrap:wrap;align-items:center">
          <input id="rw-title" placeholder="想要什麼（例：換一張月卡）" style="flex:1;min-width:160px;padding:8px;border-radius:8px;border:1px solid var(--line);background:var(--panel);color:var(--text)">
          <input id="rw-cost" type="number" placeholder="願意付多少喵喵幣" style="width:150px;padding:8px;border-radius:8px;border:1px solid var(--line);background:var(--panel);color:var(--text)">
        </div>
        <textarea id="rw-note" placeholder="想補充什麼（選填，200 字內）" rows="2"
          style="width:100%;margin-top:6px;padding:8px;border-radius:8px;border:1px solid var(--line);background:var(--panel);color:var(--text)"></textarea>
        <button onclick="reqReward(document.getElementById('rw-title').value, Number(document.getElementById('rw-cost').value||0), document.getElementById('rw-note').value)" style="margin-top:6px">送出申請</button>
      </div>
      <div style="margin-top:12px;font-size:13px">
        ${list.length ? list.map((r) => `<div style="padding:5px 0;border-bottom:1px solid var(--border)">
            <b>${esc(r.title)}</b> · ${r.cost > 0 ? fmt(r.cost) + ' 喵喵幣' : '價格由站長定'} ·
            ${r.status === 'pending' ? '<span style="color:var(--muted)">待審核</span>' : r.status === 'approved' ? '<span style="color:#5fbf7f">已通過</span>' : '<span style="color:#e06c6c">已拒絕</span>'}
            ${r.admin_note ? `<span style="color:var(--muted)">（站長：${esc(r.admin_note)}）</span>` : ''}
          </div>`).join('') : '<span style="color:var(--muted)">還沒有申請紀錄</span>'}
      </div>
    </div>`;
}

async function reqReward(title, cost, note) {
  try {
    const r = await api('POST', '/api/rewards/request', { title, note, cost });
    toast(r.message || '已送出');
    loadRewards();
  } catch (e) { toast(String(e)); }
}

// 管理員審核區
async function loadAdminRewards() {
  const box = document.getElementById('ad-rewards');
  if (!box) return;
  let d;
  try { d = await api('GET', '/api/admin/rewards'); } catch (e) {
    box.innerHTML = `<div style="color:var(--muted)">載入失敗：${esc(String(e))}</div>`; return;
  }
  const pend = (d.requests || []).filter((r) => r.status === 'pending');
  const done = (d.requests || []).filter((r) => r.status !== 'pending').slice(0, 8);
  box.innerHTML = `
    ${pend.length ? pend.map((r) => `
      <div style="padding:6px 0;border-bottom:1px solid var(--border)">
        <b>${esc(r.name)}</b> 申請「${esc(r.title)}」 ${r.cost > 0 ? '／' + fmt(r.cost) + ' 喵喵幣' : ''}
        ${r.note ? `<div style="font-size:12px;color:var(--muted)">${esc(r.note)}</div>` : ''}
        <input id="adr-${r.id}" placeholder="備註（會顯示給玩家）" style="width:100%;margin:4px 0;padding:6px;border-radius:6px;border:1px solid var(--line);background:var(--panel);color:var(--text)">
        <button onclick="decideReward(${r.id}, true)" style="font-size:12px;background:linear-gradient(180deg,#3f9c6a,#2c7a4f)">通過（並扣喵喵幣）</button>
        <button onclick="decideReward(${r.id}, false)" style="font-size:12px">拒絕</button>
      </div>`).join('') : '<div style="color:var(--muted)">沒有待審的獎勵申請</div>'}
    ${done.length ? `<div style="margin-top:8px;font-size:12px;color:var(--muted)">最近處理：${done.map((r) => `${esc(r.name)}／${esc(r.title)}／${r.status === 'approved' ? '通過' : '拒絕'}`).join('　')}</div>` : ''}`;
}

async function decideReward(id, approve) {
  const el = document.getElementById('adr-' + id);
  const note = el ? el.value : '';
  if (!confirm(approve ? '通過這筆申請並扣喵喵幣？' : '拒絕這筆申請？')) return;
  try {
    const r = await api('POST', '/api/admin/rewards/decide', { id, approve, note });
    toast(r.message || '已處理');
    loadAdminRewards();
  } catch (e) { toast(String(e)); }
}
