const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
class Element {
  constructor(){this.children=[];this.files=[];this.value='';this.disabled=false;this.hidden=false;}
  append(...items){this.children.push(...items);}
  replaceChildren(...items){this.children=items;}
  reset(){this.resetCount=(this.resetCount||0)+1;}
}
const nodes=new Map();const node=id=>{if(!nodes.has(id))nodes.set(id,new Element());return nodes.get(id);};
let releases=[],requests=[],uploadStatus=201;
const context=vm.createContext({document:{getElementById:node,createElement:()=>new Element()},FormData,console,setTimeout,clearTimeout,confirm:()=>true,fetch:async(path,options={})=>{
 requests.push({path,options});
 if(path==='/api/v1/auth/me')return {ok:false,status:401,json:async()=>({})};
 if(path==='/api/v1/firmware'&&options.method==='POST')return {ok:uploadStatus===201,status:uploadStatus};
 if(path==='/api/v1/firmware')return {ok:true,status:200,json:async()=>({releases})};
 return {ok:true,status:200,json:async()=>({})};
}});
vm.runInContext(fs.readFileSync('apps/console/assets/app.js','utf8'),context);
const flush=()=>new Promise(resolve=>setImmediate(resolve));
(async()=>{
 await flush();
 releases=[{version:'<script>alert(1)</script>',sequence:1,size:288,sha256:'a'.repeat(64),active:true}];
 await vm.runInContext('refreshFirmware()',context);
 let row=node('firmware-list').children[0];assert.equal(row.children.length,2);assert.match(row.children[0].textContent,/<script>/);assert.equal(row.children[1].textContent,'撤回发布');
 row.children[1].onclick();await flush();assert.deepEqual(JSON.parse(requests.find(r=>r.path.endsWith('/channel')).options.body),{sha256:''});
 const button=new Element();node('firmware-manifest').files=[new Blob(['{}'])];node('firmware-image').files=[new Blob([new Uint8Array(288)])];
 node('firmware-form').onsubmit({preventDefault(){},submitter:button});await flush();
 const upload=requests.find(r=>r.path==='/api/v1/firmware'&&r.options.method==='POST');assert(upload);assert.deepEqual([...upload.options.body.keys()],['manifest','image']);assert.equal(upload.options.headers,undefined);assert.equal(button.disabled,false);assert.match(node('status').textContent,/尚未发布|点击发布/);
 const resetCount=node('firmware-form').resetCount;uploadStatus=400;
 node('firmware-form').onsubmit({preventDefault(){},submitter:button});await flush();assert.equal(node('firmware-form').resetCount,resetCount);assert.equal(node('firmware-image').disabled,false);assert.match(node('status').textContent,/校验不通过/);
 console.log('OTA console multipart ordering, publication state and upload recovery: PASS');
})().catch(error=>{console.error(error);process.exitCode=1;});
