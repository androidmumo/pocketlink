/* Execute the real inline script with a minimal DOM and controlled network.
 * This tests request/state races, not browser layout or mobile captive portals. */
const fs = require('node:fs');
const vm = require('node:vm');
const assert = require('node:assert/strict');
const html = fs.readFileSync('firmware/apps/pocketlink/main/portal.html', 'utf8');
const source = html.match(/<script>([\s\S]*?)<\/script>/)[1];
const elements = Object.create(null);
for (const name of ['form','ssid','password','host','port','code','networks','scan','submit','status','feedback']) {
  elements[name] = {id:name, value:'', disabled:false, textContent:'', options:[],
    add(item){this.options.push(item)}, replaceChildren(...items){this.options=items}};
}
elements.form.querySelectorAll = () => Object.values(elements).filter(e=>!['form','status','feedback'].includes(e.id));
let status = {token:'a'.repeat(32), host:'pocketlink.mcloc.cn', port:443, ssid:'old', configured:true, busy:false, status:'idle', scan:['<img src=x onerror=alert(1)>']};
let resolveInitial;
let first = true;
let postCount = 0;
let post;
const calls = [];
const context = vm.createContext({document:{getElementById:id=>elements[id]},
  Option:function(text,value){this.text=text;this.value=value}, TextEncoder,
  setInterval(){}, fetch:async(path, options)=>{
    calls.push({path,options});
    if (options.method==='POST') {postCount++; return post(path,options)}
    if (first) {first=false; return new Promise(resolve=>{resolveInitial=()=>resolve({ok:true,json:async()=>status})})}
    return {ok:true,json:async()=>status};
  }});
function run(code){return vm.runInContext(code,context)}
function tick(){return new Promise(resolve=>setImmediate(resolve))}
(async()=>{
  run(source);
  elements.host.value='mine.example'; elements.form.oninput({target:elements.host});
  resolveInitial(); await tick();
  assert.equal(elements.host.value,'mine.example','slow first status must not overwrite edit');
  assert.equal(elements.ssid.value,'old');
  assert.equal(elements.networks.options[1].text,status.scan[0],'SSID is text, never HTML');
  const good={ssid:'家庭网络',password:'12345678',host:'pocketlink.mcloc.cn',port:443,code:''};
  context.payload=good; run('validate(payload)');
  for(const patch of [{ssid:'中'.repeat(11)},{password:'short'},{port:0},{port:1.2},{host:'https://evil'}, {code:'A'.repeat(42)+'B'}]) {
    context.payload={...good,...patch}; assert.throws(()=>run('validate(payload)'));
  }
  context.payload={...good,host:'other.example'}; assert.throws(()=>run('validate(payload)'));
  context.payload={...good,host:'other.example',code:'A'.repeat(43)}; run('validate(payload)');
  elements.password.value='retain-me'; elements.code.value='retain-code';
  let resolvePost;
  post=()=>new Promise(resolve=>{resolvePost=resolve});
  const pending=run("send('/configure',{})"); await tick();
  assert.equal(elements.submit.disabled,true);
  await run("send('/configure',{})"); assert.equal(postCount,1,'duplicate click must not send twice');
  resolvePost({ok:false,status:409,json:async()=>({error:'请重试'})}); await pending;
  assert.equal(elements.password.value,'retain-me'); assert.equal(elements.code.value,'retain-code');
  assert.equal(elements.feedback.textContent,'请重试');
  post=async()=>({ok:true,json:async()=>({})}); status={...status,busy:true};
  await run("send('/configure',{})");
  assert.equal(elements.password.value,''); assert.equal(elements.code.value,'');
  assert.equal(elements.submit.disabled,true,'accepted work remains disabled until device is idle');
  status={...status,busy:false}; await run('refresh()'); assert.equal(elements.submit.disabled,false);
  assert.equal(calls.filter(c=>c.options.method==='POST')[0].options.headers['X-PocketLink-Token'],'a'.repeat(32));
  console.log('Portal validation, edit preservation, request locking and error recovery: PASS');
})().catch(error=>{console.error(error);process.exitCode=1});
