// ---------- clawd 吉祥物（純 SVG 動畫，零外部素材）----------
// clawd = Claude 貓爪吉祥物。Anthropic 星芒 + Clay 橘 (#D97757)。
// 狀態 → 動作：鎖住=睡覺 zzz、解鎖=彈跳、底線鎖死=暈眩、捐贈=丟幣、戳=搖+說話。
//
// 「字符疊在一起」修正（2026-10-05）：
//   舊版用 .replace('/>') 事後插動畫，第一個 />} 是星芒線段的收口 ——
//   replace 把 SVG 提早關閉，後半段標記全部漏成碎字（喵？/戳/朕…）。
//   新版：clawdBase(eyes, extra, bodyAnim) 全由參數注入，零 replace。

function clawdBurst(cx, cy, r) {
  let rays = '';
  for (let i = 0; i < 8; i++) {
    const a = (Math.PI / 4) * i + Math.PI / 8;
    const x1 = cx + Math.cos(a) * r * 0.3, y1 = cy + Math.sin(a) * r * 0.3;
    const x2 = cx + Math.cos(a) * r,       y2 = cy + Math.sin(a) * r;
    rays += `<line x1="${x1.toFixed(1)}" y1="${y1.toFixed(1)}" x2="${x2.toFixed(1)}" y2="${y2.toFixed(1)}" stroke="#D97757" stroke-width="2.4" stroke-linecap="round"/>`;
  }
  return `<g><animateTransform attributeName="transform" type="rotate" dur="14s" repeatCount="indefinite" from="0 ${cx} ${cy}" to="360 ${cx} ${cy}"/>${rays}</g>`;
}

function clawdBase(eyes, extra, bodyAnim) {
  return `<svg viewBox="0 0 120 120" width="120" height="120" xmlns="http://www.w3.org/2000/svg" style="overflow:visible">
  <g>
    ${bodyAnim || ''}
    <path d="M84 94 q20 0 18 -22 q-2 -16 -16 -12" fill="none" stroke="#E8B98F" stroke-width="9" stroke-linecap="round">
      <animate attributeName="d" dur="3.2s" repeatCount="indefinite"
        values="M84 94 q20 0 18 -22 q-2 -16 -16 -12; M84 94 q22 4 12 -24 q-6 -12 -18 -6; M84 94 q20 0 18 -22 q-2 -16 -16 -12"/>
    </path>
    <ellipse cx="56" cy="84" rx="28" ry="22" fill="#D97757"/>
    <circle cx="56" cy="84" r="12" fill="#F5E6D3"/>
    ${clawdBurst(56, 84, 7.5)}
    <path d="M34 56 l-5 -19 l17 9 z" fill="#D97757"/>
    <path d="M78 56 l5 -19 l-17 9 z" fill="#D97757"/>
    <circle cx="56" cy="54" r="23" fill="#D97757"/>
    <circle cx="41" cy="61" r="4.5" fill="#F1957A" opacity=".65"/>
    <circle cx="71" cy="61" r="4.5" fill="#F1957A" opacity=".65"/>
    ${eyes}
    <path d="M53 63 l3 3 l3 -3" fill="none" stroke="#7C2D12" stroke-width="2" stroke-linecap="round"/>
    <path d="M56 66 q0 3.5 -3.5 3.5 M56 66 q0 3.5 3.5 3.5" fill="none" stroke="#7C2D12" stroke-width="1.6" stroke-linecap="round"/>
    <ellipse cx="38" cy="98" rx="7" ry="5" fill="#E8B98F"/>
    <ellipse cx="74" cy="98" rx="7" ry="5" fill="#E8B98F"/>
    ${extra || ''}
  </g></svg>`;
}

const EYES_OPEN = `
  <circle cx="47" cy="52" r="3.2" fill="#3B1E12">
    <animate attributeName="ry" values="3.2;3.2;0.4;3.2" keyTimes="0;0.9;0.93;1" dur="4.5s" repeatCount="indefinite"/>
  </circle>
  <circle cx="65" cy="52" r="3.2" fill="#3B1E12">
    <animate attributeName="ry" values="3.2;3.2;0.4;3.2" keyTimes="0;0.9;0.93;1" dur="4.5s" repeatCount="indefinite"/>
  </circle>`;

const EYES_SLEEP = `
  <path d="M44 52 q3 3 6 0 M62 52 q3 3 6 0" fill="none" stroke="#3B1E12" stroke-width="2.2" stroke-linecap="round"/>`;

const EYES_DIZZY = `
  <g stroke="#3B1E12" stroke-width="1.8" fill="none">
    <circle cx="47" cy="52" r="3.2"/>
    <circle cx="65" cy="52" r="3.2"/>
    <path d="M44 49 l6 6 M50 49 l-6 6 M62 49 l6 6 M68 49 l-6 6" stroke-width="1.4"/>
  </g>`;

// 1b) 夢想（0-25%）：坐著抬頭看星星，偶爾眨眼
function clawdDream() {
  return clawdBase(EYES_OPEN, `
    <g opacity=".8">
      <text x="84" y="26" font-size="10" fill="#E8B98F">✧<animate attributeName="opacity" values=".2;.8;.2" dur="2.6s" repeatCount="indefinite"/></text>
      <text x="94" y="38" font-size="7" fill="#B08968">✧<animate attributeName="opacity" values=".8;.2;.8" dur="3.2s" repeatCount="indefinite"/></text>
      <text x="76" y="18" font-size="8" fill="#D97757" opacity=".5"><animate attributeName="opacity" values=".5;.9;.5" dur="4s" repeatCount="indefinite"/>…</text>
    </g>`,
    `<animateTransform attributeName="transform" type="translate" values="0 2; 0 0; 0 2" dur="4.2s" repeatCount="indefinite"/>`);
}

// 1c) 希望（25-50%）：站起來仰望 + 小愛心
function clawdHopeful() {
  return clawdBase(EYES_OPEN, `
    <g>
      <text x="88" y="34" font-size="11" opacity=".7">♥<animate attributeName="opacity" values=".3;.9;.3" dur="1.9s" repeatCount="indefinite"/></text>
      <text x="10" y="44" font-size="9" fill="#B08968" opacity=".5">✧<animate attributeName="opacity" values=".5;.9;.5" dur="2.4s" repeatCount="indefinite"/></text>
    </g>`,
    `<animateTransform attributeName="transform" type="translate" values="0 0; 0 -3; 0 0" dur="2.2s" repeatCount="indefinite" calcMode="spline" keySplines="0.4 0 0.6 1;0.4 0 0.6 1"/>`);
}

// 1d) 興奮（50-75%）：耳朵抖動 + 轉圈小跑 + ⚡
function clawdExcited() {
  return clawdBase(EYES_OPEN, `
    <g>
      <text x="86" y="24" font-size="12" fill="#F5C542">⚡<animate attributeName="opacity" values="1;.3;1" dur="0.7s" repeatCount="indefinite"/></text>
      <text x="12" y="34" font-size="10" fill="#E8B98F" opacity=".6">✧</text>
      <animateTransform attributeName="transform" type="rotate" values="0 56 100; 3 56 100; -3 56 100; 0 56 100" dur="0.8s" repeatCount="indefinite"/>
    </g>`,
    `<animateTransform attributeName="transform" type="translate" values="0 0; 0 -5; 0 0" dur="0.55s" repeatCount="indefinite"/>`);
}

// 1e) 開市派對（開市中 1 小時）：墨鏡 + 幣雨 + 蹦迪
function clawdParty() {
  const coins = [];
  for (let i = 0; i < 5; i++) {
    const x = 10 + Math.random() * 100;
    coins.push(`<circle cx="${x.toFixed(0)}" cy="-10" r="4" fill="#F5C542" stroke="#B7860B" stroke-width="1.5">
      <animate attributeName="cy" values="-10;125" dur="${(1.6 + Math.random()).toFixed(1)}s" begin="${(Math.random() * 1.2).toFixed(1)}s" repeatCount="indefinite"/>
    </circle>`);
  }
  return `<svg viewBox="0 0 120 120" width="120" height="120" xmlns="http://www.w3.org/2000/svg" style="overflow:visible">
  <animateTransform attributeName="transform" type="translate" values="0 0; 0 -12; 0 0" dur="0.5s" repeatCount="indefinite" calcMode="spline" keySplines="0.3 0 0.7 1;0.3 0 0.7 1"/>
  <g>
    <path d="M84 94 q20 0 18 -22 q-2 -16 -16 -12" fill="none" stroke="#E8B98F" stroke-width="9" stroke-linecap="round">
      <animate attributeName="d" dur="0.7s" repeatCount="indefinite"
        values="M84 94 q20 0 18 -22 q-2 -16 -16 -12; M84 94 q24 -6 10 -26 q-8 -10 -20 -2; M84 94 q20 0 18 -22 q-2 -16 -16 -12"/>
    </path>
    <ellipse cx="56" cy="84" rx="28" ry="22" fill="#D97757"/>
    <circle cx="56" cy="84" r="12" fill="#F5E6D3"/>
    ${clawdBurst(56, 84, 7.5)}
    <path d="M34 56 l-5 -19 l17 9 z" fill="#D97757"/>
    <path d="M78 56 l5 -19 l-17 9 z" fill="#D97757"/>
    <circle cx="56" cy="54" r="23" fill="#D97757"/>
    <!-- 墨鏡 -->
    <g><rect x="38" y="46" width="16" height="10" rx="3" fill="#1a1a1a"/><rect x="60" y="46" width="16" height="10" rx="3" fill="#1a1a1a"/><path d="M54 50 h6 M38 50 l-6 -3 M76 50 l6 -3" stroke="#1a1a1a" stroke-width="2"/>
      <path d="M40 48 h12 M62 48 h12" stroke="#66ccff" stroke-width="1.4" opacity=".7"><animate attributeName="opacity" values=".7;.2;.7" dur="0.6s" repeatCount="indefinite"/></path></g>
    <path d="M50 64 q6 6 12 0" fill="none" stroke="#7C2D12" stroke-width="2.2" stroke-linecap="round"/>
    <ellipse cx="38" cy="98" rx="7" ry="5" fill="#E8B98F"/>
    <ellipse cx="74" cy="98" rx="7" ry="5" fill="#E8B98F"/>
    ${coins.join('')}
    <text x="8" y="20" font-size="12" opacity=".8">🎉</text>
    <text x="98" y="18" font-size="12" opacity=".8">🎉<animate attributeName="opacity" values=".8;.3;.8" dur="0.9s" repeatCount="indefinite"/></text>
  </g></svg>`;
}

// 1) 睡覺（未解鎖）：呼吸起伏 + zzz
function clawdSleep() {
  return clawdBase(EYES_SLEEP, `
    <g font-size="11" fill="#B08968" font-weight="bold">
      <text x="82" y="32" opacity="0">z<animate attributeName="opacity" values="0;1;0" dur="3s" repeatCount="indefinite"/></text>
      <text x="92" y="22" opacity="0">z<animate attributeName="opacity" values="0;1;0" dur="3s" begin="1s" repeatCount="indefinite"/></text>
      <text x="101" y="13" opacity="0">z<animate attributeName="opacity" values="0;1;0" dur="3s" begin="2s" repeatCount="indefinite"/></text>
    </g>`,
    `<animateTransform attributeName="transform" type="translate" values="0 1; 0 -1; 0 1" dur="3.6s" repeatCount="indefinite"/>`);
}

// 2) 開心（解鎖）：原地彈跳 + ✦
function clawdHappy() {
  return clawdBase(EYES_OPEN, `
    <g>
      <text x="6" y="30" font-size="14" fill="#E8B98F" opacity=".5">✦<animate attributeName="opacity" values=".5;1;.5" dur="1.3s" repeatCount="indefinite"/></text>
      <text x="102" y="26" font-size="11" fill="#D97757" opacity=".5">✦<animate attributeName="opacity" values="1;.5;1" dur="1.6s" repeatCount="indefinite"/></text>
    </g>`,
    `<animateTransform attributeName="transform" type="translate" values="0 0; 0 -9; 0 0" dur="0.9s" repeatCount="indefinite" calcMode="spline" keySplines="0.4 0 0.6 1;0.4 0 0.6 1"/>`);
}

// 3) 暈眩（底線鎖死）：X 眼 + 搖晃 + 🌀
function clawdDizzy() {
  return clawdBase(EYES_DIZZY, `
    <g>
      <text x="14" y="98" font-size="15" opacity=".9">🌀</text>
      <text x="96" y="100" font-size="15" opacity=".9">🌀<animate attributeName="opacity" values=".9;.2;.9" dur="0.8s" repeatCount="indefinite"/></text>
    </g>`,
    `<animateTransform attributeName="transform" type="rotate" values="-2 56 110; 2 56 110; -2 56 110" dur="1.1s" repeatCount="indefinite"/>`);
}

// 4) 丟幣（捐贈 flash）：喵喵幣拋進池子
function clawdCoin() {
  return clawdBase(EYES_OPEN, `
    <g>
      <circle cx="24" cy="26" r="9" fill="#F5C542" stroke="#B7860B" stroke-width="2">
        <animate attributeName="cx" values="24;50;50" keyTimes="0;.55;1" dur="1s" repeatCount="indefinite"/>
        <animate attributeName="cy" values="26;80;80" keyTimes="0;.55;1" dur="1s" repeatCount="indefinite"/>
        <animate attributeName="opacity" values="1;1;0" keyTimes="0;.55;1" dur="1s" repeatCount="indefinite"/>
      </circle>
      <text font-size="9" font-weight="bold" fill="#8a6503">
        <animate attributeName="x" values="20;46;46" keyTimes="0;.55;1" dur="1s" repeatCount="indefinite"/>
        <animate attributeName="y" values="30;84;84" keyTimes="0;.55;1" dur="1s" repeatCount="indefinite"/>
        <animate attributeName="opacity" values="1;1;0" keyTimes="0;.55;1" dur="1s" repeatCount="indefinite"/>
        喵
      </text>
      <path d="M40 90 q10 -12 24 -4" stroke="#F5C542" stroke-width="3" fill="none" opacity="0">
        <animate attributeName="opacity" values="0;.9;0" keyTimes="0;.55;1" dur="1s" repeatCount="indefinite"/>
      </path>
    </g>`);
}

// 依池子狀態挑吉祥物（5h 撞牆 → 睡覺等重置）
// ★ 進度分段（user 2026-10-06）：0-25-50-75% 各有動態+語錄
function clawdFor(p) {
  if (!p) return clawdDream();
  if (p.open_until && p.unlocked) return clawdParty();
  if (p.floor_lock) return clawdDizzy();
  if (p.unlocked)   return clawdHappy();
  // 未開市：按進度
  const prog = p.threshold > 0 && p.threshold < 4503599627370496 ? (p.pool_chips / p.threshold) : 0;
  if (prog >= 0.75) return clawdExcited();   // 75%+ 就快到了
  if (prog >= 0.50) return clawdHopeful();   // 過半
  if (prog >= 0.25) return clawdDream();     // 有進度
  return clawdSleep();
}
function clawdMood(p) {
  if (!p) return 'sleeping';
  if (p.unlocked && p.open_until) return 'party';
  if (p.floor_lock) return 'dizzy';
  if (p.unlocked) return 'happy';
  const prog = p.threshold > 0 && p.threshold < 4503599627370496 ? (p.pool_chips / p.threshold) : 0;
  if (prog >= 0.75) return 'excited';
  if (prog >= 0.50) return 'hopeful';
  if (prog >= 0.25) return 'dream';
  return 'sleeping';
}

// ---------- 戳戳語錄（依池子狀態分組）----------
const CLAWD_LINES = {
  sleeping: [
    'zzz… 池子還空著啦…',
    '（打到呼嚕）嗯…5 小時…',
    '喵？捐滿了再叫我。',
    '睡給你看，反正 Claude 鎖著。',
    '（尾巴抖了一下）別戳…沒額度…',
  ],
  dream: [
    '（做白日夢）要是有 Claude 可以用…',
    '池子有一點點了喵…繼續。',
    '☆ 想像著大家用 Claude 的樣子…',
    '（抬頭看天）今天會有人捐嗎？',
    '25% 了喵……還早，但在動。',
  ],
  hopeful: [
    '過半了！有希望的感覺喵！',
    '（坐直）再一點點…再一點點…',
    '♥ 大家都在存錢的感覺真好。',
    '你也在等開市嗎？我也是。',
    '50% —— 續攤的人在哪～',
  ],
  excited: [
    '⚡快了快了！衝啊喵！',
    '（原地打轉）75%了吶！！',
    '差一點差一點！叫大家來捐！',
    '（小跑心跳 180）開市就在眼前！',
    '再一腳就到門檻了喵！！！',
  ],
  party: [
    '🎉 開市啦！去用 Claude 喵！！',
    '（蹦迪中）一小時嗨起來——',
    '墨鏡戴上，額度燒起！',
    '喵喵幣雨～淋到就是用到的～',
    '別戳了快去用 Claude 啦！',
  ],
  happy: [
    '開啦開啦！去用 Claude 喵！',
    '（星芒轉速 200%）今天的我是發電機。',
    '戳什麼戳，快去寫 code！',
    '池子暖暖的，額度香香的。',
    '再戳我就把你的 prompt 吞掉喵！',
  ],
  dizzy: [
    '頭好暈…額度見底了…',
    '（轉圈圈）7 天窗口在轉…',
    '救…救額度…',
    '剩不到 15% 了…',
    '別轉了別轉了…',
  ],
  waiting: [
    '快了快了，就差一點喵。',
    '（盯著門檻）差多少自己看啦。',
    '再捐一點嘛，就差臨門一腳。',
    '（坐直）門快開了喔。',
  ],
};

let __clawdLastLine = -1;
function clawdSay(line) {
  const el = document.getElementById('clawd-bubble');
  if (!el) return;
  el.textContent = line;
  el.style.display = 'block';
  el.style.animation = 'none'; void el.offsetWidth; el.style.animation = 'clawdPop .25s ease-out';
  clearTimeout(el.__hideT);
  el.__hideT = setTimeout(() => { el.style.display = 'none'; }, 3200);
}

function clawdPick(arr) {
  let i; do { i = Math.floor(Math.random() * arr.length); } while (arr.length > 1 && i === __clawdLastLine);
  __clawdLastLine = i;
  return arr[i];
}

// 戳：依狀態給語錄 + 搖一下
function clawdPoke() {
  const p = window.__aipoolState;
  const box = document.getElementById('aipool-mascot');
  const mood = clawdMood(p);
  if (box) {
    box.style.animation = 'none'; void box.offsetWidth;
    box.style.animation = 'clawdShake .4s ease-out';
  }
  clawdSay(clawdPick(CLAWD_LINES[mood]));
}

// 依狀態注入吉祥物（對話框走 document flow，放吉祥物下方 —— 不絕對定位、不疊字）
function clawdMount(p) {
  window.__aipoolState = p;
  const box = document.getElementById('aipool-mascot');
  if (!box) return;
  box.innerHTML = clawdFor(p) + `
    <div id="clawd-bubble" style="display:none;margin-top:6px;width:120px;
      background:var(--panel2,var(--card));border:1px solid var(--gold,var(--border));border-radius:10px;padding:6px 8px;
      font-size:12px;color:var(--text,var(--foreground));box-shadow:0 4px 14px rgba(0,0,0,.25);z-index:5;line-height:1.5"></div>`;
}

// 🎁 神秘彩蛋 popup（不在任何說明文字出現）：本輪個人累計捐過「門檻 10%」→ 捐贈當下彈一次
function clawdEggPopup(justDonated) {
  const p = window.__aipoolState;
  if (!p || !p.threshold || p.threshold >= 4503599627370496) return;
  const line = p.threshold * 0.10;
  const mine = (p.my_contrib || 0) + (justDonated || 0); // my_contrib 是本輪（後端已改 round）
  if (mine >= line && !window.__clawdEggShown) {
    window.__clawdEggShown = true;
    const el = document.createElement('div');
    el.style.cssText = 'position:fixed;top:74px;right:18px;background:linear-gradient(135deg,#2a2415,#3a3018);border:1px solid var(--gold);color:var(--gold);padding:12px 18px;border-radius:12px;z-index:400;font-size:13.5px;box-shadow:0 10px 30px rgba(0,0,0,.5);animation:clawdPop .3s ease-out;max-width:280px';
    el.textContent = '🎁 clawd 對你悄悄話：「你好像做了件好事…收市的時候，可能會有驚喜喵。」';
    document.body.appendChild(el);
    setTimeout(() => { el.style.transition = 'opacity .5s'; el.style.opacity = '0'; setTimeout(() => el.remove(), 500); }, 5000);
  }
}