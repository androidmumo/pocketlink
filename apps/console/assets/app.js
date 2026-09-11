"use strict";
const $ = id => document.getElementById(id);
let expiryTimer;
function status(text) { $("status").textContent = text; }
function loggedIn(yes) {
  $("login").hidden = yes; $("workspace").hidden = !yes;
  if (!yes) { clearTimeout(expiryTimer); $("pair-code").textContent = ""; $("pair-result").hidden = true; $("devices").replaceChildren(); }
}
async function api(path, method = "GET", data) {
  const response = await fetch("/api/v1/" + path, {method, credentials:"same-origin", headers: data ? {"Content-Type":"application/json"} : {}, body: data ? JSON.stringify(data) : undefined});
  const result = await response.json();
  if (!response.ok) {
    if (response.status === 401) loggedIn(false);
    const messages = {401:"登录已失效或凭证不正确，请重新登录。", 403:"请求来源不匹配，请从配置的 HTTPS 地址打开本页。", 429:"请求过于频繁，请稍后重试。", 409:"已达到配对或设备数量上限。", 503:"服务暂不可用，请稍后重试。"};
    throw new Error(messages[response.status] || "操作失败，请检查输入后重试。");
  }
  return result;
}
async function refresh() {
  const {devices} = await api("devices");
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
}
async function perform(button, action) {button.disabled = true; status(""); try {await action();} catch(error) {status(error.message);} finally {button.disabled = false;}}
$("login-form").onsubmit = event => {event.preventDefault(); perform(event.submitter, async () => {try {await api("auth/login", "POST", {password:$("password").value});} finally {$("password").value="";} loggedIn(true); await refresh();});};
$("pair-form").onsubmit = event => {event.preventDefault(); perform(event.submitter, async () => {const pair = await api("pairings", "POST", {name:$("device-name").value}); clearTimeout(expiryTimer); $("pair-result").hidden=false; $("pair-code").textContent=pair.code; $("pair-expiry").textContent="到期时间："+new Date(pair.expires_at*1000).toLocaleTimeString(); expiryTimer=setTimeout(()=>{$("pair-code").textContent="配对码已过期，请重新生成。";},Math.max(0,pair.expires_at*1000-Date.now()));});};
$("logout").onclick = () => perform($("logout"), async () => {await api("auth/logout", "POST"); loggedIn(false);});
$("refresh").onclick = () => perform($("refresh"), refresh);
(async()=>{try {await api("auth/me"); loggedIn(true); await refresh();} catch(error) {loggedIn(false); if (!error.message.startsWith("登录")) status(error.message);}})();
