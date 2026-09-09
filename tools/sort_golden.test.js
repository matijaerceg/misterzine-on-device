const {test}=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const os=require('node:os');
const path=require('node:path');
const {execFileSync}=require('node:child_process');
test('updated ties use arrival batches; debut ties still use titles',()=>{
 const tmp=fs.mkdtempSync(path.join(os.tmpdir(),'mz-golden-'));
 try {
  const site=path.join(tmp,'site'), releases=path.join(site,'docs','releases'),out=path.join(tmp,'out');
  fs.mkdirSync(releases,{recursive:true});
  fs.writeFileSync(path.join(releases,'index.html'),'const CORE_NAMES = {\n};');
  const data=[
   {k:'old',title:'A old',core:'a',updated:'2026-09-08',date:'2020-01-01'},
   {k:'new',title:'Z new',core:'z',updated:'2026-09-08',date:'2020-01-01',b:2},
   {k:'middle',title:'M middle',core:'m',updated:'2026-09-08',date:'2020-01-01',b:1}
  ];
  fs.writeFileSync(path.join(releases,'data.json'),JSON.stringify(data));
  fs.writeFileSync(path.join(releases,'meta.json'),'{}');
  execFileSync(process.execPath,[path.join(__dirname,'sort_golden.js'),site,out]);
  const golden=JSON.parse(fs.readFileSync(path.join(out,'sort_golden.json')));
  assert.deepEqual(golden.updated,['new','middle','old']);
  assert.deepEqual(golden.debut,['old','middle','new']);
  assert.deepEqual(JSON.parse(fs.readFileSync(path.join(out,'data.json'))),data);
 } finally {
  if(path.dirname(tmp)!==path.resolve(os.tmpdir()) || !path.basename(tmp).startsWith('mz-golden-')) throw Error('unexpected temp path');
  fs.rmSync(tmp,{recursive:true});
 }
});
