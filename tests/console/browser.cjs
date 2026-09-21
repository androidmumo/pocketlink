// Run against a disposable HTTPS loopback server, never production.
const assert=require('node:assert/strict');
const fs=require('node:fs');
const {chromium}=require(process.env.POCKETLINK_PLAYWRIGHT||'playwright');
const origin=process.env.POCKETLINK_TEST_ORIGIN||'https://127.0.0.1:3443';
assert.equal(new URL(origin).hostname,'127.0.0.1');
const password=fs.readFileSync(process.env.POCKETLINK_TEST_PASSWORD_FILE,'utf8').trim();
const output=process.env.POCKETLINK_SCREENSHOTS||'/private/tmp/pocketlink-ui-test';
(async()=>{
 const browser=await chromium.launch({executablePath:process.env.POCKETLINK_CHROME||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',headless:true});
 try{
 const context=await browser.newContext({ignoreHTTPSErrors:true,viewport:{width:1440,height:1000}});
 const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 const waitText=async(selector,text)=>page.waitForFunction(({selector,text})=>document.querySelector(selector).textContent.includes(text),{selector,text});
 await page.goto(origin);await page.screenshot({path:output+'/login-desktop.png',fullPage:true});
 await page.click('#mode-admin');await page.fill('#password',password);await page.click('#login-form button');await page.waitForSelector('#workspace:not([hidden])');await page.waitForTimeout(300);
 await page.screenshot({path:output+'/overview-desktop.png',fullPage:true});
 await page.click('#nav-invitations');await page.click('#create-invitation');await page.waitForSelector('#invitation-result:not([hidden])');
 const code=await page.textContent('#invitation-code');assert.equal(code.length,43);
 await page.reload();await page.waitForSelector('#workspace:not([hidden])');await page.click('#nav-invitations');await page.getByRole('button',{name:'查看 / 复制'}).first().click();await page.waitForSelector('#invitation-result:not([hidden])');assert.equal(await page.textContent('#invitation-code'),code);
 await context.grantPermissions(['clipboard-read','clipboard-write']);await page.click('#copy-invitation');assert.equal(await page.evaluate(()=>navigator.clipboard.readText()),code);

 await page.click('#logout');await page.waitForSelector('#login:not([hidden])');await page.click('#mode-register');
 const username='browser_'+Date.now();
 await page.fill('#register-username',username);await page.fill('#register-password','local-browser-test-password');await page.fill('#register-confirm','local-browser-test-password');await page.fill('#registration-code',code);await page.click('#register-form button');await waitText('#status','账号已创建');
 await page.fill('#password','local-browser-test-password');await page.click('#login-form button');await page.waitForSelector('#workspace:not([hidden])');assert(await page.locator('#nav-firmware').isHidden());
 await page.click('#nav-devices');await page.fill('#device-name','口袋一号');await page.click('#pair-form button');await page.waitForSelector('#pair-result:not([hidden])');
 const pairingCode=await page.textContent('#pair-code');const response=await context.request.post(origin+'/api/v1/device/pair',{data:{code:pairingCode,sn:'browser-device-'+Date.now()}});assert.equal(response.status(),201);const device=await response.json();
 await page.click('#refresh');await waitText('#devices','口袋一号');await page.click('#nav-rooms');await page.fill('#room-name','周末出游');await page.click('#room-form button');await page.waitForSelector('#room-detail:not([hidden])');

 assert(await page.locator('#room-select').isHidden());await page.click('#room-picker');await page.waitForSelector('#room-options:not([hidden])');await page.keyboard.press('Home');await page.keyboard.press('Enter');await page.waitForSelector('#room-detail[hidden]',{state:'attached'});
 await page.focus('#room-picker');await page.keyboard.press('ArrowDown');await page.keyboard.press('End');await page.keyboard.press('Enter');await page.waitForSelector('#room-detail:not([hidden])');
 await page.click('#room-picker');await page.keyboard.press('Escape');assert(await page.locator('#room-options').isHidden());
 const member=page.locator('#room-members input').first();await member.check();await waitText('#status','已加入房间');
 await page.fill('#message-text','周六下午一起出发，记得带上 PocketLink。');await page.click('#send-text');await waitText('#status','已保存');await page.waitForSelector('.message-item');
 const inbox=await context.request.get(origin+'/api/v1/device/inbox',{headers:{Authorization:'Bearer '+device.credential}});assert.equal(inbox.status(),200);const msg=(await inbox.json()).messages[0];assert.equal(msg.text,'周六下午一起出发，记得带上 PocketLink。');
 const ack=await context.request.post(origin+'/api/v1/device/ack',{headers:{Authorization:'Bearer '+device.credential},data:{message_id:msg.id,state:'read'}});assert.equal(ack.status(),200);
 await page.locator('.message-item button').first().click();await waitText('#message-history','已读');
 await page.screenshot({path:output+'/rooms-desktop.png',fullPage:true});
 await page.click('#invite-room');await page.waitForSelector('#invitation-result:not([hidden])');const roomCode=await page.textContent('#invitation-code');assert.equal(roomCode.length,43);assert(await page.locator('#create-invitation').isHidden());
 await page.setViewportSize({width:390,height:844});await page.click('#nav-rooms');await page.click('#room-picker');await page.waitForSelector('#room-options:not([hidden])');const pickerBounds=await page.locator('#room-options').boundingBox();assert(pickerBounds.x>=0&&pickerBounds.x+pickerBounds.width<=390&&pickerBounds.y>=0&&pickerBounds.y+pickerBounds.height<=844);await page.locator('#room-options [role=option]').last().click();await page.screenshot({path:output+'/rooms-mobile.png',fullPage:true});
 assert(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth),'mobile overflow');
 await page.click('#logout');await page.waitForSelector('#login:not([hidden])');await page.screenshot({path:output+'/login-mobile.png',fullPage:true});
 assert(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth),'login overflow');
 // A second account accepts the invitation through the UI, sees no old history,
 // and can leave without gaining owner controls.
 const adminContext=await browser.newContext({ignoreHTTPSErrors:true});
 const adminLogin=await adminContext.request.post(origin+'/api/v1/auth/login',{headers:{Origin:origin},data:{username:'admin',password}});assert.equal(adminLogin.status(),200);
 const issued=await adminContext.request.post(origin+'/api/v1/invitations',{headers:{Origin:origin},data:{kind:'registration'}});assert.equal(issued.status(),201);const secondCode=(await issued.json()).code;
 await adminContext.close();
 await page.click('#mode-register');await page.fill('#register-username','friend_'+Date.now());await page.fill('#register-password','local-browser-test-password');await page.fill('#register-confirm','local-browser-test-password');await page.fill('#registration-code',secondCode);await page.click('#register-form button');await waitText('#status','账号已创建');await page.fill('#password','local-browser-test-password');await page.click('#login-form button');await page.waitForSelector('#workspace:not([hidden])');
 await page.click('#nav-rooms');await page.fill('#join-code',roomCode);await page.click('#join-form button');await waitText('#status','已加入房间');assert(await page.locator('#archive-room').isHidden());assert(await page.locator('#invite-room').isHidden());assert.equal(await page.locator('.message-item').count(),0);
 page.once('dialog',d=>d.accept());await page.click('#leave-room');await waitText('#status','已退出房间');assert(await page.locator('#room-detail').isHidden());
 assert.deepEqual(errors,[]);
 console.log('Browser: admin invite, registration, ordinary login, role UI, device pairing, room message/receipt, room invite, logout, desktop/mobile layout PASS');
 await context.close();
 }finally{await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
