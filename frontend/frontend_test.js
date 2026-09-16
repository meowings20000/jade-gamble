// frontend_test.js — node:assert + vm smoke tests (no deps).
const assert = require('assert');
const fs = require('fs');
const path = require('path');
const vm = require('vm');

const dir = __dirname;
const html = fs.readFileSync(path.join(dir, 'index.html'), 'utf8');
const appSrc = fs.readFileSync(path.join(dir, 'app.js'), 'utf8');
const renderSrc = fs.readFileSync(path.join(dir, 'stone-render.js'), 'utf8');

// 1. every id referenced by $('#...') in app.js exists in index.html
const idRefs = [...appSrc.matchAll(/\$\('#([\w-]+)'\)/g)].map(m => m[1]);
const uniqueIds = [...new Set(idRefs)];
for (const id of uniqueIds) {
  assert.ok(html.includes(`id="${id}"`), `app.js references #${id} but index.html lacks it`);
}
console.log(`✓ all ${uniqueIds.length} referenced ids exist in index.html`);

// also template-created ids used via querySelector after injection
for (const id of ['m-close', 'scr-cv', 'scr-acc', 'scr-hint', 'scr-sell', 'scr-close',
  'pol-mult', 'pol-ladder', 'pol-adv', 'pol-cash', 'pol-close']) {
  assert.ok(appSrc.includes(`id="${id}"`), `dynamic id ${id} must be created in app.js templates`);
}
console.log('✓ dynamic modal ids are defined in templates');

// 2. no external http(s) resource loads (single-container, offline)
const extRefs = [...html.matchAll(/https?:\/\/[^"' ]+/g)].map(m => m[0]);
assert.ok(extRefs.length === 0, `index.html loads external resources: ${extRefs}`);
console.log('✓ no external resource loads');

// 3. app.js parses as a script
new vm.Script(appSrc, { filename: 'app.js' });
new vm.Script(renderSrc, { filename: 'stone-render.js' });
console.log('✓ app.js and stone-render.js parse');

// 4. StoneRender core: deterministic thumbnail (canvas 2d stub)
const calls = [];
const ctxStub = new Proxy({}, {
  get(t, prop) {
    if (prop === 'createRadialGradient' || prop === 'createLinearGradient') {
      return () => ({ addColorStop: () => calls.push('stop') });
    }
    return () => {};
  },
  set() { return true; },
});
const sandbox = {
  window: {},
  document: {
    createElement: () => ({ getContext: () => ctxStub, width: 0, height: 0 }),
  },
  Math, console,
};
vm.createContext(sandbox);
vm.runInContext(renderSrc, sandbox);
const SR = sandbox.window.StoneRender;
assert.ok(SR && SR.thumbnail && SR.revealView && SR.VARIETY_PALETTE, 'StoneRender exports');
assert.ok(Object.keys(SR.VARIETY_PALETTE).length === 10, '10 variety palettes');

// rng determinism
const r1 = SR.rngFrom(123), r2 = SR.rngFrom(123);
const a1 = [r1(), r1(), r1()], a2 = [r2(), r2(), r2()];
assert.deepStrictEqual(a1, a2, 'same seed → same sequence');
console.log('✓ renderer deterministic, 10 palettes');

// 5. API surface sanity: every fetch path in app.js matches backend routes
const routes = fs.readFileSync(path.join(dir, '..', 'backend', 'api', 'api.go'), 'utf8');
const appPaths = [...new Set([...appSrc.matchAll(/api\('(?:GET|POST)',\s*'(\/api\/[\w/?=-]+)'/g)].map(m => m[1]))];
assert.ok(appPaths.length >= 20, `expected ~27 client API paths, found ${appPaths.length}`);
for (const p of appPaths) {
  const bare = p.split('?')[0].replace('/api/', '');
  assert.ok(routes.includes(bare), `app.js calls ${p} but no route "${bare}" in api.go`);
}
console.log(`✓ all ${appPaths.length} client API paths exist on server`);
console.log('\nALL FRONTEND TESTS PASSED');