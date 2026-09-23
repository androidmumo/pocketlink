// Isolated loopback test accounts only; never run against production.
const fs=require('fs'),assert=require('assert/strict');
const {chromium}=require(process.env.POCKETLINK_PLAYWRIGHT||'playwright');
const origin=process.env.POCKETLINK_TEST_ORIGIN||'https://127.0.0.1:3443';assert.equal(new URL(origin).hostname,'127.0.0.1');
(async()=>{const browser=await chromium.launch({executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',headless:true});try{
 const admin=await browser.newContext({ignoreHTTPSErrors:true,viewport:{width:1440,height:1000}}),user=await browser.newContext({ignoreHTTPSErrors:true});
 const p=await admin.newPage(),q=await user.newPage(),errors=[];p.on('pageerror',e=>errors.push(e.message));q.on('pageerror',e=>errors.push(e.message));p.on('dialog',d=>d.accept());
 await p.goto(origin);await p.click('#mode-admin');await p.fill('#password',fs.readFileSync(process.env.POCKETLINK_TEST_PASSWORD_FILE,'utf8').trim());await p.click('#login-form button');await p.waitForSelector('#workspace:not([hidden])');
 const api=async(context,path,data,expected=200)=>{const r=await context.request.fetch(origin+'/api/v1/'+path,{method:data?'POST':'GET',headers:{Origin:origin},data});assert.equal(r.status(),expected,await r.text());return r.json()};
 const name='manage_'+Date.now(),password='initial-password-for-browser',next='changed-password-for-browser';
 const invitation=await api(admin,'invitations',{kind:'registration'},201);await api(user,'auth/register',{username:name,password,code:invitation.code},201);
 await q.goto(origin);await q.fill('#username',name);await q.fill('#password',password);await q.click('#login-form button');await q.waitForSelector('#workspace:not([hidden])');assert(await q.locator('#nav-users').isHidden());await api(user,'users',null,403);
 await p.click('#nav-users');await p.click('#refresh-users');await p.fill('#user-search',name);const card=p.locator('.user-card').filter({hasText:name});await card.getByRole('button',{name:'强制退出',exact:true}).click();await p.waitForFunction(()=>document.querySelector('#status').textContent.includes('强制退出已完成'));await api(user,'auth/me',null,401);
 await api(user,'auth/login',{username:name,password});await card.getByRole('button',{name:'重置密码',exact:true}).click();await p.fill('#user-new-password',next);await p.click('#user-password-form button');await p.waitForSelector('#user-password-panel',{state:'hidden'});assert.equal(await p.inputValue('#user-new-password'),'');await api(user,'auth/me',null,401);await api(user,'auth/login',{username:name,password},401);await api(user,'auth/login',{username:name,password:next});
 await card.getByRole('button',{name:'禁用账号',exact:true}).click();await card.getByRole('button',{name:'恢复账号',exact:true}).waitFor();await api(user,'auth/me',null,401);await api(user,'auth/login',{username:name,password:next},401);
 await p.screenshot({path:'/private/tmp/pocketlink-ui-test/users-desktop.png'});await p.setViewportSize({width:390,height:844});await p.screenshot({path:'/private/tmp/pocketlink-ui-test/users-mobile.png'});assert(await p.evaluate(()=>document.documentElement.scrollWidth<=innerWidth));
 await card.getByRole('button',{name:'恢复账号',exact:true}).click();await card.getByRole('button',{name:'禁用账号',exact:true}).waitFor();await api(user,'auth/login',{username:name,password:next});
 await p.fill('#user-search','admin');assert.equal(await p.locator('.user-card').getByRole('button').count(),0);await p.click('#logout');await p.waitForSelector('#login:not([hidden])');assert.equal(await p.locator('#user-list').textContent(),'');assert.equal(await p.inputValue('#user-new-password'),'');assert.deepEqual(errors,[]);
 console.log('User management: admin-only UI/API, search, force logout, reset, disable/enable, protected admin, logout cleanup and mobile layout PASS');
}finally{await browser.close()}})().catch(e=>{console.error(e);process.exitCode=1});
