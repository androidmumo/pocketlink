"use strict";
const $ = id => document.getElementById(id);
let expiryTimer;
let deviceRows = [];
let roomRows = [];
let roomEpoch = 0;
let historyCursor = 0;
let historyEpoch = 0;
let pendingSend = null;
function status(text) { $("status").textContent = text; }
function loggedIn(yes) {
  $("login").hidden = yes; $("workspace").hidden = !yes;
  if (!yes) { clearTimeout(expiryTimer); $("pair-code").textContent = ""; $("pair-result").hidden = true; $("devices").replaceChildren(); $("room-detail").hidden=true; $("room-select").replaceChildren(); $("message-text").value=""; pendingSend=null; roomEpoch++; }
}
async function api(path, method = "GET", data) {
  const response = await fetch("/api/v1/" + path, {method, credentials:"same-origin", headers: data ? {"Content-Type":"application/json"} : {}, body: data ? JSON.stringify(data) : undefined});
  const result = await response.json();
  if (!response.ok) {
    if (response.status === 401) loggedIn(false);
    const messages = {401:"登录已失效或凭证不正确，请重新登录。", 403:"请求来源不匹配，请从配置的 HTTPS 地址打开本页。", 429:"请求过于频繁，请稍后重试。", 409:"已达到配对或设备数量上限。", 503:"服务暂不可用，请稍后重试。"};
    const codes = {empty_room:"房间没有有效成员，请先添加设备。", idempotency_conflict:"重试标识对应的消息内容不同，请刷新页面并核对历史。", capacity_reached:"已达到存储或数量上限。", invalid_request:"请检查输入：文字最多 200 字符，不能为空或包含控制字符。"};
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
}
async function perform(button, action) {button.disabled = true; status(""); try {await action();} catch(error) {status(error.message);} finally {button.disabled = false;}}
$("login-form").onsubmit = event => {event.preventDefault(); perform(event.submitter, async () => {try {await api("auth/login", "POST", {password:$("password").value});} finally {$("password").value="";} loggedIn(true); await refresh();});};
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
  $("room-detail").hidden=!id; $("message-history").replaceChildren(); $("room-members").replaceChildren(); historyCursor=0;
  if (!id) return;
  const room=roomRows.find(r=>r.id===id); const archived=room.archived_at!==null;
  $("message-form").hidden=archived; $("archive-room").disabled=archived;
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
    const li=document.createElement("li");li.className="message-item";const title=document.createElement("small");title.textContent="#"+message.id+" · "+new Date(message.created_at*1000).toLocaleString();const text=document.createElement("p");text.className="message-body";text.textContent=message.text;
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
(async()=>{try {await api("auth/me"); loggedIn(true); await refresh();} catch(error) {loggedIn(false); if (!error.message.startsWith("登录")) status(error.message);}})();
