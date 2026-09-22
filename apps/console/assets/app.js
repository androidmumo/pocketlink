"use strict";
const $ = id => document.getElementById(id);
let expiryTimer;
let currentUser = null;
let sessionEpoch = 0;
let authMode = "login";
let currentView = "overview";
let deviceRows = [];
let roomRows = [];
let roomEpoch = 0;
let historyCursor = 0;
let historyEpoch = 0;
let pendingSend = null;
let noticeTimer;
function status(text) { clearTimeout(noticeTimer); $("status").textContent = text; if(text)noticeTimer=setTimeout(()=>{$("status").textContent="";},10000); }
function loggedIn(yes, identity) {
  if(yes || currentUser!==null) sessionEpoch++;
  if (identity) currentUser = {...identity.user, role:identity.role};
  if (!yes) { globalThis.PocketVoice?.disconnect();$("room-action-panel").hidden=true;$("room-invitation-list").replaceChildren();$("room-invitation-code").textContent="";$("room-invitation-result").hidden=true; currentUser=null; deviceRows=[]; roomRows=[]; $("invitation-code").textContent=""; $("invitation-result").hidden=true; $("invitation-list").replaceChildren(); }
  if (yes && currentUser) {
    const admin=currentUser.role==="admin";
    $("account-name").textContent=currentUser.username;
    $("account-role").textContent=admin?"管理员":"普通用户";
    $("devices-nav-label").textContent=admin?"设备管理":"我的设备";
    $("avatar").textContent=currentUser.username[0].toUpperCase();
    $("welcome-title").textContent="你好，"+currentUser.username;
    $("nav-firmware").hidden=!admin; $("nav-invitations").hidden=!admin; $("create-invitation").hidden=!admin;
    showView("overview");
  }
  $("login").hidden = yes; $("workspace").hidden = !yes;
  if (!yes) { $("firmware-form").reset(); $("firmware-list").replaceChildren(); clearTimeout(expiryTimer); $("pair-code").textContent = ""; $("pair-result").hidden = true; $("devices").replaceChildren(); $("room-detail").hidden=true; $("room-select").replaceChildren(); $("message-text").value=""; pendingSend=null; roomEpoch++; }
}
async function api(path, method = "GET", data) {
  const epoch=sessionEpoch;
  const response = await fetch("/api/v1/" + path, {method, credentials:"same-origin", headers: data ? {"Content-Type":"application/json"} : {}, body: data ? JSON.stringify(data) : undefined});
  const result = await response.json();
  if(epoch!==sessionEpoch)throw Object.assign(new Error("会话已切换"),{stale:true});
  if (!response.ok) {
    if (response.status === 401) loggedIn(false);
    const messages = {401:"登录已失效或凭证不正确，请重新登录。", 403:"没有执行此操作的权限，或页面来源不匹配。", 404:"内容不存在，或你已失去访问权限。", 429:"请求过于频繁，请稍后重试。", 409:"已达到配对或设备数量上限。", 503:"服务暂不可用，请稍后重试。"};
    const codes = {admin_required:"此操作仅限管理员。",invalid_invitation:"邀请码无效、已使用或已过期。",invalid_registration:"用户名需为字母开头的 3–32 位字母、数字或下划线，密码至少 12 位。",username_unavailable:"该用户名不可用，请换一个。",empty_room:"房间没有有效成员，请先添加设备。", idempotency_conflict:"重试标识对应的消息内容不同，请刷新页面并核对历史。", capacity_reached:"已达到存储或数量上限。", invalid_request:"请检查输入：文字最多 200 字符，不能为空或包含控制字符。"};
    throw new Error(codes[result.error] || messages[response.status] || "操作失败，请检查输入后重试。");
  }
  return result;
}
async function refresh() {
  const {devices} = await api("devices");
  deviceRows = devices;
  $("devices").replaceChildren();
  if (!devices.length) { const empty = document.createElement("li"); empty.textContent = "还没有设备。先生成配对码，再完成设备端配对。"; $("devices").append(empty); }
  for (const device of devices) {
    const item = document.createElement("li"); const text = document.createElement("span");
    text.textContent = `${device.name} · ${device.sn} · ${device.revoked_at === null ? "有效" : "已撤销"}`; item.append(text);
    if (device.revoked_at === null) {
      const revoke = document.createElement("button"); revoke.className = "secondary"; revoke.textContent = "撤销凭证";
      revoke.onclick = () => perform(revoke, async () => {
        if (!confirm(`撤销「${device.name}」的访问凭证？`)) return;
        await api("devices/" + encodeURIComponent(device.id), "DELETE"); await refresh(); status("设备凭证已撤销。");
      }); item.append(revoke);
    }
    $("devices").append(item);
  }
  await refreshRooms();
  if(currentUser?.role==="admin") await refreshFirmware();
  await refreshInvitations();
  $("device-count").textContent=String(devices.filter(d=>d.revoked_at===null).length);
  $("room-count").textContent=String(roomRows.filter(r=>r.archived_at===null).length);
}
async function perform(button, action) {button.disabled = true; status(""); try {await action();} catch(error) {if(!error.stale)status(error.message);} finally {button.disabled = false;}}
$("login-form").onsubmit = event => {event.preventDefault(); perform(event.submitter, async () => {let identity;try {identity=await api("auth/login", "POST", {username:authMode==="admin"?"admin":$("username").value,password:$("password").value});} finally {$("password").value="";} loggedIn(true,identity); await refresh();});};
$("pair-form").onsubmit = event => {event.preventDefault(); perform(event.submitter, async () => {const pair = await api("pairings", "POST", {name:$("device-name").value}); clearTimeout(expiryTimer); $("pair-result").hidden=false; $("pair-code").textContent=pair.code; $("pair-expiry").textContent="到期时间："+new Date(pair.expires_at*1000).toLocaleTimeString(); expiryTimer=setTimeout(()=>{$("pair-code").textContent="配对码已过期，请重新生成。";},Math.max(0,pair.expires_at*1000-Date.now()));});};
$("logout").onclick = () => perform($("logout"), async () => {await api("auth/logout", "POST"); loggedIn(false);});
$("refresh").onclick = () => perform($("refresh"), refresh);
function selectedRoom() { return $("room-select").value; }
async function refreshRooms() {
  const previous = selectedRoom();
  const {rooms} = await api("rooms"); roomRows = rooms;
  $("room-select").replaceChildren();
  const empty = document.createElement("option"); empty.value=""; empty.textContent="请选择房间"; $("room-select").append(empty);
  for (const room of rooms) { const option=document.createElement("option"); option.value=room.id; option.textContent=room.name+(room.archived_at===null?"":"（已归档）"); $("room-select").append(option); }
  $("room-select").value=rooms.some(r=>r.id===previous)?previous:"";
  await loadRoom();
}
async function loadRoom() {
  const id=selectedRoom(); const epoch=++roomEpoch;
  globalThis.PocketVoice?.roomChanged(id);
  $("invite-room").disabled=true;$("room-action-panel").hidden=true;$("room-invitation-code").textContent="";$("room-invitation-result").hidden=true;
  $("room-detail").hidden=!id; $("room-empty").hidden=!!id; $("message-history").replaceChildren(); $("room-members").replaceChildren(); historyCursor=0;
  if (!id) return;
  const room=roomRows.find(r=>r.id===id); const archived=room.archived_at!==null; $("voice-panel").hidden=archived;
  const owner=currentUser && (currentUser.role==="admin"||room.owner_id===currentUser.id);
  $("message-form").hidden=archived; $("archive-room").disabled=archived; $("archive-room").hidden=!owner;
  $("invite-room").disabled=!owner||archived;$("invite-room").title=owner?"邀请朋友加入当前房间":"仅房主可以邀请朋友";globalThis.PocketVoice?.setRoom(id,!archived); $("leave-room").hidden=owner||archived;
  await loadRoomUsers(id,epoch,owner);
  if(epoch!==roomEpoch)return;
  const {device_ids}=await api("rooms/"+encodeURIComponent(id)+"/members");
  if (epoch!==roomEpoch) return;
  const active=deviceRows.filter(d=>d.revoked_at===null);
  if (!active.length) $("room-members").textContent="暂无有效设备，请先完成设备配对。";
  for (const device of active) {
    const label=document.createElement("label"); label.className="member-choice";
    const check=document.createElement("input"); check.type="checkbox"; check.checked=device_ids.includes(device.id); check.disabled=archived;
    const name=document.createElement("span"); name.textContent=device.name+" · "+device.sn; label.append(check,name); $("room-members").append(label);
    check.onchange=async()=>{const adding=check.checked;check.disabled=true;try{await api("rooms/"+encodeURIComponent(id)+"/members/"+encodeURIComponent(device.id),adding?"PUT":"DELETE");status(adding?"已加入房间。":"已移出房间，旧消息停止投递。");}catch(error){check.checked=!adding;status(error.message);}finally{check.disabled=archived;}};
  }
  await loadHistory(false);
}
async function loadHistory(older) {
  const id=selectedRoom();if (!id) return;const epoch=roomEpoch;const historyRequest=++historyEpoch;
  const page=await api("rooms/"+encodeURIComponent(id)+"/messages"+(older&&historyCursor?"?before="+historyCursor:""));
  if (epoch!==roomEpoch||historyRequest!==historyEpoch) return;
  if (!older) $("message-history").replaceChildren();
  historyCursor=page.next_before;$("older-messages").hidden=!historyCursor;
  if (!older&&!page.messages.length) {const li=document.createElement("li");li.textContent="这个房间还没有消息。";$("message-history").append(li);}
  for(const message of page.messages){
    const li=document.createElement("li");li.className="message-item";const title=document.createElement("small");title.textContent=(message.sender||"房间消息")+" · #"+message.id+" · "+new Date(message.created_at*1000).toLocaleString();const text=document.createElement("p");text.className="message-body";text.textContent=message.text;
    const receipts=document.createElement("div");const button=document.createElement("button");button.className="secondary";button.textContent="查看 / 刷新回执";
    button.onclick=()=>perform(button,async()=>{const data=await api("messages/"+message.id+"/receipts");receipts.replaceChildren();for(const receipt of data.receipts){const row=document.createElement("p");const state=receipt.read_at!==null?"已读":receipt.delivered_at!==null?"已接收":"待接收";row.textContent=receipt.name+"："+state+(receipt.withdrawn_at!==null?"（后续投递已撤回）":"");receipts.append(row);}});
    li.append(title,text,button,receipts);$("message-history").append(li);
  }
}
$("room-form").onsubmit=event=>{event.preventDefault();perform(event.submitter,async()=>{const room=await api("rooms","POST",{name:$("room-name").value});$("room-name").value="";await refreshRooms();$("room-select").value=room.id;await loadRoom();status("房间已创建，请勾选成员。");});};
$("room-select").onchange=()=>{loadRoom().catch(error=>status(error.message));};
$("archive-room").onclick=()=>perform($("archive-room"),async()=>{const id=selectedRoom();if(!id||!confirm("归档后停止投递且不能再发送；消息记录保留。确定归档？"))return;await api("rooms/"+encodeURIComponent(id),"DELETE");await refreshRooms();status("房间已归档。");});
$("message-text").oninput=()=>{$("text-count").textContent=Array.from($("message-text").value).length+" / 200";};
$("message-form").onsubmit=event=>{event.preventDefault();perform(event.submitter,async()=>{
  const room=selectedRoom(),text=$("message-text").value;if(!room||!text.trim()||Array.from(text).length>200)throw new Error("请选择房间并填写 1 至 200 个字符。");
  if(!pendingSend||pendingSend.room!==room||pendingSend.text!==text)pendingSend={room,text,request_id:crypto.randomUUID()};
  $("message-text").readOnly=true;$("room-select").disabled=true;
  try {const result=await api("rooms/"+encodeURIComponent(room)+"/messages","POST",{request_id:pendingSend.request_id,text});pendingSend=null;$("message-text").value="";$("text-count").textContent="0 / 200";status("消息 #"+result.id+" 已保存，等待设备接收。");try{await loadHistory(false);}catch{status("消息已保存，刷新记录失败，请手动刷新。");}}
  finally{$("message-text").readOnly=false;$("room-select").disabled=false;}
});};
$("refresh-messages").onclick=()=>perform($("refresh-messages"),()=>loadHistory(false));
$("older-messages").onclick=()=>perform($("older-messages"),()=>loadHistory(true));
async function refreshFirmware() {
  const {releases}=await api("firmware");
  $("firmware-list").replaceChildren();
  if(!releases.length) $("firmware-list").textContent="尚未上传固件。";
  for(const release of releases) {
    const item=document.createElement("li"), label=document.createElement("span");
    label.textContent=`${release.version} · 序号 ${release.sequence} · ${Math.ceil(release.size/1024)} KiB · ${release.active?"已发布":"未发布"}`;
    const dates=document.createElement("small");dates.className="release-dates";
    dates.textContent="上传时间："+new Date(release.created_at*1000).toLocaleString()+" · 最近发布时间："+(release.published_at?new Date(release.published_at*1000).toLocaleString():"未记录");
    label.append(dates);item.append(label);
    const publish=document.createElement("button");publish.className="secondary";publish.textContent=release.active?"撤回发布":"发布";
    publish.onclick=()=>perform(publish,async()=>{
      if(!confirm(release.active?"撤回后停止新的下载，已经开始的升级可能继续。确认撤回？":`发布 ${release.version}？设备检查后仍需按键确认升级。`))return;
      await api("firmware/channel","PUT",{sha256:release.active?"":release.sha256});await refreshFirmware();status("发布设置已保存；这不表示设备已升级。");
    });item.append(publish);
    if(!release.active){const remove=document.createElement("button");remove.className="secondary";remove.textContent="删除";remove.onclick=()=>perform(remove,async()=>{if(!confirm(`删除 ${release.version} 的升级包？其序号不能重复使用。`))return;await api("firmware/"+release.sha256,"DELETE");await refreshFirmware();});item.append(remove);}
    $("firmware-list").append(item);
  }
}
$("firmware-form").onsubmit=event=>{event.preventDefault();perform(event.submitter,async()=>{
  const manifest=$("firmware-manifest").files[0],image=$("firmware-image").files[0];
  if(!manifest||!image||manifest.size>4096||image.size<288||image.size>3*1024*1024)throw new Error("请选择签名清单和不超过 3 MiB 的应用固件。");
  const epoch=sessionEpoch;
  const form=new FormData();form.append("manifest",manifest);form.append("image",image);
  $("firmware-manifest").disabled=true;$("firmware-image").disabled=true;
  try {
    const response=await fetch("/api/v1/firmware",{method:"POST",credentials:"same-origin",body:form});
    if(epoch!==sessionEpoch)return;
    if(response.status===401)loggedIn(false);
    if(!response.ok)throw new Error(response.status===409?"序号必须递增，同一固件不能重复签名上传；请检查已有版本。":response.status===400?"签名、目标设备或固件校验不通过。":response.status===429?"请求过多或存储已满，请稍后重试或删除旧版本。":"上传失败，请确认登录和服务器状态。");
    $("firmware-form").reset();await refreshFirmware();status("升级包已校验并保存，点击发布后设备才能发现更新。");
  } finally {$("firmware-manifest").disabled=false;$("firmware-image").disabled=false;}
});};

(async()=>{try {const identity=await api("auth/me"); loggedIn(true,identity); await refresh();} catch(error) {if(error.stale)return; loggedIn(false); if (!error.message.startsWith("登录")) status(error.message);}})();

const viewLabels={overview:["概览","设备相连，消息随行。"],devices:["我的设备","让每一台设备，都有归属。"],rooms:["房间与消息","与朋友共享一个频道。"],invitations:["注册邀请码","邀请朋友注册 PocketLink。"],firmware:["固件管理","为设备带来新的能力。"]};
function showView(name){
 if((name==="firmware"||name==="invitations")&&currentUser?.role!=="admin")name="overview";
 if(name!=="rooms")globalThis.PocketVoice?.disconnect();
 currentView=name;
 for(const key of Object.keys(viewLabels)){$("view-"+key).hidden=key!==name;$("nav-"+key).className="nav-item"+(key===name?" active":"");}
 $("page-name").textContent=viewLabels[name][0];$("page-title").textContent=viewLabels[name][1];
}
for(const key of Object.keys(viewLabels))$("nav-"+key).onclick=()=>showView(key);
$("overview-add").onclick=()=>showView("devices");
function setAuthMode(mode){
 authMode=mode;const registering=mode==="register",admin=mode==="admin";
 $("login-form").hidden=registering;$("register-form").hidden=!registering;
 $("username").hidden=admin;$("username").required=!admin;$("username-label").hidden=admin;
 $("password").value="";$("register-password").value="";$("register-confirm").value="";
 $("auth-title").textContent=registering?"创建你的空间":admin?"管理员入口":"登录你的空间";
 $("auth-intro").textContent=admin?"使用服务器配置的管理员密码登录。":"管理设备、加入房间，让消息抵达。";
 $("auth-help").textContent=admin?"管理员密码由服务器文件配置，注册账号不会获得管理员权限。":"还没有账号？请向管理员获取注册邀请码。";
 for(const key of ["login","register","admin"])$("mode-"+key).className=key===mode?"active":"";
}
for(const mode of ["login","register","admin"])$("mode-"+mode).onclick=()=>setAuthMode(mode);
$("register-form").onsubmit=event=>{event.preventDefault();perform(event.submitter,async()=>{
 if($("register-password").value!==$("register-confirm").value)throw new Error("两次输入的密码不一致。");
 const username=$("register-username").value.trim();
 await api("auth/register","POST",{username,password:$("register-password").value,code:$("registration-code").value.trim()});
 $("register-form").reset();setAuthMode("login");$("username").value=username;status("账号已创建，请使用新账号登录。");
});};
async function copyCode(id){const value=$(id).textContent;if(!value||value.includes("过期"))throw new Error("请先生成有效邀请码。");try{await navigator.clipboard.writeText(value);status("已复制，请私下分享给对应的人。");}catch{throw new Error("无法自动复制，请长按或选中邀请码手动复制。");}}
$("copy-pair").onclick=()=>perform($("copy-pair"),()=>copyCode("pair-code"));
$("copy-invitation").onclick=()=>perform($("copy-invitation"),()=>copyCode("invitation-code"));
function openRoomAction(action){
 $("room-action-panel").hidden=false;$("room-form").hidden=action!=="create";$("join-form").hidden=action!=="join";$("room-invite-panel").hidden=action!=="invite";
 $("room-action-title").textContent=action==="create"?"创建新的房间":action==="join"?"用邀请码加入房间":"邀请朋友加入 · "+(roomRows.find(r=>r.id===selectedRoom())?.name||"");
 if(action==="create")$("room-name").focus();else if(action==="join")$("join-code").focus();
}
$("open-create-room").onclick=()=>openRoomAction("create");$("open-join-room").onclick=()=>openRoomAction("join");$("close-room-action").onclick=()=>{$("room-action-panel").hidden=true;};
function displayInvitation(entry,code){
 const prefix=entry.kind==="room"?"room-invitation":"invitation";
 $(prefix+"-result").hidden=false;$(prefix+"-code").textContent=code;$(prefix+"-expiry").textContent="有效至 "+new Date(entry.expires_at*1000).toLocaleString()+" · 一次有效";
 if(entry.kind==="registration")$("invitation-title").textContent="账号注册邀请码";
}
async function createInvitation(kind,room){
 const epoch=roomEpoch;const data=await api("invitations","POST",{kind,room_id:room||""});
 await refreshInvitations();if(kind==="room"&&epoch!==roomEpoch)return;
 displayInvitation({...data,kind},data.code);if(kind==="registration")showView("invitations");status("邀请码已生成，可再次查看和复制。");
}
$("copy-room-invitation").onclick=()=>perform($("copy-room-invitation"),()=>copyCode("room-invitation-code"));
$("create-invitation").onclick=()=>perform($("create-invitation"),()=>createInvitation("registration"));
$("invite-room").onclick=()=>perform($("invite-room"),async()=>{openRoomAction("invite");await refreshInvitations();});
$("new-room-invitation").onclick=()=>perform($("new-room-invitation"),()=>createInvitation("room",selectedRoom()));
async function refreshInvitations(){
 const epoch=roomEpoch;const {invitations}=await api("invitations");if(epoch!==roomEpoch)return;
 for(const kind of ["registration","room"]){
  const prefix=kind==="room"?"room-invitation":"invitation",list=$(prefix+"-list");list.replaceChildren();
  const entries=invitations.filter(e=>e.kind===kind&&(kind!=="room"||e.room_id===selectedRoom()));
  if(!entries.length)list.textContent="还没有生成过邀请码。";
  for(const entry of entries){
   const li=document.createElement("li"),text=document.createElement("span");
   const state=entry.revoked_at!==null?"已撤销":entry.used_at!==null?"已使用":entry.expires_at*1000<=Date.now()?"已过期":"待使用";
   text.textContent=state+" · "+new Date(entry.expires_at*1000).toLocaleDateString()+" 到期";li.append(text);
   if(state==="待使用"){
    if(entry.code_available){const view=document.createElement("button");view.className="secondary";view.textContent="查看 / 复制";
     view.onclick=()=>perform(view,async()=>{const data=await api("invitations/"+entry.id+"/code");if(kind==="room"&&epoch!==roomEpoch)return;displayInvitation(entry,data.code);$(kind==="room"?"copy-room-invitation":"copy-invitation").focus();});li.append(view);
    }else{const legacy=document.createElement("small");legacy.textContent="旧邀请码未保存原码；如已遗失，请撤销后重新生成。";li.append(legacy);}
    const revoke=document.createElement("button");revoke.className="secondary";revoke.textContent="撤销";revoke.onclick=()=>perform(revoke,async()=>{await api("invitations/"+entry.id,"DELETE");await refreshInvitations();if(kind==="room"&&epoch!==roomEpoch)return;$(prefix+"-result").hidden=true;$(prefix+"-code").textContent="";status("邀请码已撤销。");});li.append(revoke);
   }list.append(li);
  }
 }
}
$("join-form").onsubmit=event=>{event.preventDefault();perform(event.submitter,async()=>{
 const result=await api("rooms/join","POST",{code:$("join-code").value.trim()});$("join-code").value="";await refreshRooms();$("room-select").value=result.room_id;await loadRoom();status("已加入房间，请选择自己的接收设备。加入前的消息不会显示。");
});};
async function loadRoomUsers(room,epoch,owner){
 const {users}=await api("rooms/"+encodeURIComponent(room)+"/users");if(epoch!==roomEpoch)return;$("room-users").replaceChildren();
 const roomOwner=roomRows.find(r=>r.id===room)?.owner_id;
 for(const user of users){const li=document.createElement("li"),label=document.createElement("span");label.textContent=user.username+(user.id===roomOwner?" · 房主":"");li.append(label);
 if(owner&&user.id!==roomOwner){const remove=document.createElement("button");remove.className="secondary";remove.textContent="移除";remove.onclick=()=>perform(remove,async()=>{if(!confirm("移除该用户及其房间内设备？"))return;await api("rooms/"+encodeURIComponent(room)+"/users/"+encodeURIComponent(user.id),"DELETE");await loadRoom();});li.append(remove);}
 $("room-users").append(li);}
}
$("leave-room").onclick=()=>perform($("leave-room"),async()=>{if(!confirm("退出房间并移除自己的接收设备？"))return;await api("rooms/"+encodeURIComponent(selectedRoom())+"/users/"+encodeURIComponent(currentUser.id),"DELETE");await refreshRooms();status("已退出房间。");});
