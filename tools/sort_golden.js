#!/usr/bin/env node
// Produces testdata/sort_golden.json: the exact row order the misterzine site
// shows for "Last updated" (default) and "Debut" sorts, computed with the
// site's own comparator and its own CORE_NAMES table, so the Go port is tested
// against the browser's collation rather than against a reading of it.
//
//   node tools/sort_golden.js ../misterzine testdata
//
// Mirrors docs/releases/index.html: coreLabel (1498-1504), the ingest()
// sole-title index (4029-4043) and the apply() comparator (2062-2091).
'use strict';
const fs = require('fs');
const path = require('path');

const site = process.argv[2] || '../misterzine';
const out = process.argv[3] || 'testdata';
const html = fs.readFileSync(path.join(site, 'docs', 'releases', 'index.html'), 'utf8');
const DATA = JSON.parse(fs.readFileSync(path.join(site, 'docs', 'releases', 'data.json'), 'utf8'));

// Lift CORE_NAMES out of the page verbatim and evaluate it as JS.
const m = html.match(/const CORE_NAMES = \{[\s\S]*?\n\};/);
if (!m) throw new Error('CORE_NAMES not found');
const CORE_NAMES = new Function(m[0] + '\nreturn CORE_NAMES;')();

const coreSoleTitle = {};
{
  const seen = {};
  for (const r of DATA) {
    if (!r.core) continue;
    if (!(r.core in seen)) seen[r.core] = r.title;
    else if (seen[r.core] !== r.title) seen[r.core] = null;
  }
  for (const c in seen) if (seen[c]) coreSoleTitle[c] = seen[c];
}

function coreLabel(core) {
  if (!core) return '';
  if (CORE_NAMES[core]) return CORE_NAMES[core];
  if (coreSoleTitle[core]) return coreSoleTitle[core];
  return core.replace(/([a-z0-9])([A-Z])/g, '$1 $2').replace(/^./, c => c.toUpperCase());
}

function sorted(sortKey, sortDir) {
  const view = DATA.slice();
  const keyOf = x => (x[sortKey] || '');
  view.sort((a, b) => {
    const av = keyOf(a), bv = keyOf(b);
    const ae = av === '', be = bv === '';
    if (ae !== be) return ae ? 1 : -1;
    const cmp = sortDir * av.localeCompare(bv, undefined, { numeric: true, sensitivity: 'base' });
    if (cmp) return cmp;
    if (sortKey === 'updated') {
      const cc = coreLabel(a.core).localeCompare(coreLabel(b.core), undefined, { numeric: true, sensitivity: 'base' });
      if (cc) return cc;
    }
    return a.title.localeCompare(b.title, undefined, { numeric: true, sensitivity: 'base' });
  });
  return view.map(r => r.k);
}

// A title-only ordering exercises the collation on every title at once.
const byTitle = DATA.slice().sort((a, b) =>
  a.title.localeCompare(b.title, undefined, { numeric: true, sensitivity: 'base' })).map(r => r.k);

const labels = {};
for (const r of DATA) if (r.core && !(r.core in labels)) labels[r.core] = coreLabel(r.core);

const golden = {
  rows: DATA.length,
  updated: sorted('updated', -1),
  debut: sorted('date', -1),
  title: byTitle,
  coreLabels: labels,
};
fs.mkdirSync(out, { recursive: true });
fs.writeFileSync(path.join(out, 'sort_golden.json'), JSON.stringify(golden));
fs.copyFileSync(path.join(site, 'docs', 'releases', 'data.json'), path.join(out, 'data.json'));
fs.copyFileSync(path.join(site, 'docs', 'releases', 'meta.json'), path.join(out, 'meta.json'));
console.log(`sort_golden.json: ${DATA.length} rows, ${Object.keys(labels).length} core labels; data.json + meta.json copied`);
