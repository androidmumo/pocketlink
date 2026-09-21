const assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
class Element{constructor(){this.children=[];this.value='';}append(...items){this.children.push(...items)}replaceChildren(...items){this.children=items}reset(){}}
const nodes=new Map(),node=id=>{if(!nodes.has(id))nodes.set(id,new Element());return nodes.get(id)};
let complete;
const context=vm.createContext({document:{getElementById:node,createElement:()=>new Element()},setTimeout:()=>0,clearTimeout:()=>{},fetch:async(path)=>{
 if(path.endsWith('/devices'))return new Promise(resolve=>{complete=resolve});
 return {ok:false,status:401,json:async()=>({})};
}});
vm.runInContext(fs.readFileSync('apps/console/assets/app.js','utf8'),context);
(async()=>{
 await new Promise(resolve=>setImmediate(resolve));
 for(const status of [200,401]){
  vm.runInContext('loggedIn(true,{user:{id:"alice",username:"alice"},role:"user"})',context);
  const pending=vm.runInContext('refresh().catch(error=>error)',context);
  vm.runInContext('loggedIn(false);loggedIn(true,{user:{id:"bob",username:"bob"},role:"user"})',context);
  complete({ok:status===200,status,json:async()=>({devices:[{id:'private-alice-device',name:'private',sn:'secret',revoked_at:null}]})});
  const error=await pending;assert.equal(error.stale,true);
  assert.equal(vm.runInContext('currentUser.id',context),'bob');assert.equal(vm.runInContext('deviceRows.length',context),0);assert.equal(node('workspace').hidden,false);
 }
 console.log('Account switch discards old data and stale unauthorized responses: PASS');
})().catch(e=>{console.error(e);process.exitCode=1});
