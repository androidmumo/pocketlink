(()=>{
 const get=id=>document.getElementById(id),talk=get('voice-talk'),connect=get('voice-connect'),notice=get('voice-status');
 let socket,context,media,capture,activeRoom='',generation=0,held=false,pending=0,request=0,stream=0,rx=0,rxSequence=0,sequence=0,nextPlay=0,limitTimer,requestTimer;
 const sources=new Set();
 const say=text=>{notice.textContent=text;};
 const send=value=>{if(socket?.readyState===WebSocket.OPEN)socket.send(JSON.stringify(value));};
 function stopPlayback(){for(const source of sources){try{source.stop();}catch{}}sources.clear();nextPlay=0;}
 function release(){held=false;pending=0;clearTimeout(limitTimer);clearTimeout(requestTimer);capture?.port.postMessage({stream:0});if(stream)send({type:'release',stream});stream=0;talk.dataset.speaking='false';talk.textContent='按住讲话';}
 function disconnect(text='已退出对讲；语音不会保存。'){
  generation++;release();const old=socket;socket=null;old?.close();media?.getTracks().forEach(t=>t.stop());media=null;capture?.disconnect();capture=null;stopPlayback();context?.close().catch(()=>{});context=null;activeRoom='';rx=rxSequence=0;talk.disabled=true;connect.disabled=false;connect.textContent='连接对讲';say(text);
 }
 async function join(){
  if(socket||context){disconnect();return;}
  const room=get('room-select').value;if(!room||get('voice-panel').hidden){say('请先选择有效房间。');return;}
  const epoch=++generation;connect.disabled=true;say('正在请求麦克风权限…');
  try{
   const Audio=window.AudioContext||window.webkitAudioContext;if(!Audio||!navigator.mediaDevices?.getUserMedia)throw new Error('当前浏览器不支持网页对讲，请使用 HTTPS 和新版浏览器。');
   const audio=new Audio();context=audio;await audio.resume();
   const microphone=await navigator.mediaDevices.getUserMedia({audio:{channelCount:1,echoCancellation:true,noiseSuppression:true,autoGainControl:true}});
   if(epoch!==generation){microphone.getTracks().forEach(t=>t.stop());return;}media=microphone;
   await audio.audioWorklet.addModule('/assets/voice-worklet.js');if(epoch!==generation)return;
   capture=new AudioWorkletNode(audio,'pocketlink-capture');const input=audio.createMediaStreamSource(microphone);input.connect(capture);const mute=audio.createGain();mute.gain.value=0;capture.connect(mute).connect(audio.destination);
   capture.port.onmessage=e=>{if(epoch!==generation)return;capture.port.postMessage('ack');if(e.data.stream!==stream||!held||!stream||socket?.readyState!==WebSocket.OPEN)return;if(socket.bufferedAmount>648*8){disconnect('网络发送积压，对讲已停止。');return;}const pcm=new Int16Array(e.data.pcm),packet=new ArrayBuffer(648),view=new DataView(packet);view.setUint32(0,stream);view.setUint32(4,++sequence);for(let i=0;i<320;i++)view.setInt16(8+i*2,pcm[i],true);socket.send(packet);};
   activeRoom=room;socket=new WebSocket(location.origin.replace(/^http/,'ws')+'/api/v1/voice/'+encodeURIComponent(room),'pocketlink.voice.v1');socket.binaryType='arraybuffer';
   socket.onopen=()=>{if(epoch!==generation)return;talk.disabled=false;connect.disabled=false;connect.textContent='退出对讲';say('已连接，按住讲话；每次最多 29 秒。');};
   socket.onclose=()=>{if(epoch===generation)disconnect('对讲连接已断开，请重新连接。');};socket.onerror=()=>{if(epoch===generation)disconnect('无法连接对讲，请检查房间权限与网络。');};
   socket.onmessage=e=>{
    if(epoch!==generation)return;
    if(typeof e.data==='string'){
     let v;try{v=JSON.parse(e.data);}catch{disconnect('对讲协议错误。');return;}
     if(v.type==='grant'){
      if(!held||v.request_id!==pending){send({type:'release',stream:v.stream});return;}
      clearTimeout(requestTimer);pending=0;stream=v.stream;sequence=0;stopPlayback();capture.port.postMessage({stream});talk.dataset.speaking='true';talk.textContent='正在讲话 · 松开结束';say('正在向房间讲话');limitTimer=setTimeout(()=>{release();say('已到单次讲话时限，请松开后重试。');},29000);
     }else if(v.type==='busy'){if(v.request_id===pending){release();say('有人正在讲话，请稍后再按。');}}
     else if(v.type==='floor'){rx=v.stream;rxSequence=0;stopPlayback();if(stream&&stream!==rx)release();if(!held)say(rx?'正在收听：'+v.sender:'房间空闲，按住讲话。');}
     return;
    }
    if(!(e.data instanceof ArrayBuffer)||e.data.byteLength!==648||stream||!context)return;
    const v=new DataView(e.data),serial=v.getUint32(4);if(v.getUint32(0)!==rx||!rx||serial<=rxSequence)return;rxSequence=serial;
    if(context.state!=='running'){disconnect('音频已暂停，请重新连接。');return;}
    if(nextPlay-context.currentTime>.16)stopPlayback();
    const buffer=context.createBuffer(1,320,16000),data=buffer.getChannelData(0);for(let i=0;i<320;i++)data[i]=v.getInt16(8+2*i,true)/32768;
    const source=context.createBufferSource();source.buffer=buffer;source.connect(context.destination);sources.add(source);source.onended=()=>{sources.delete(source);source.disconnect();};const when=Math.max(context.currentTime+.01,nextPlay);source.start(when);nextPlay=when+.02;
   };
  }catch(error){if(epoch===generation)disconnect(error.name==='NotAllowedError'?'麦克风权限未授予，可允许后重新连接。':error.message);}
 }
 function press(){if(held||talk.disabled||socket?.readyState!==WebSocket.OPEN)return;held=true;stopPlayback();request=(request+1)>>>0;if(!request)request=1;pending=request;send({type:'request',request_id:pending});say('正在申请讲话…');requestTimer=setTimeout(()=>{release();say('申请超时，请重试。');},3000);}
 talk.addEventListener('pointerdown',e=>{e.preventDefault();talk.setPointerCapture(e.pointerId);press();});
 for(const event of ['pointerup','pointercancel','lostpointercapture'])talk.addEventListener(event,release);
 talk.addEventListener('contextmenu',e=>e.preventDefault());talk.addEventListener('keydown',e=>{if((e.key===' '||e.key==='Enter')&&!e.repeat){e.preventDefault();press();}});talk.addEventListener('keyup',e=>{if(e.key===' '||e.key==='Enter'){e.preventDefault();release();}});talk.addEventListener('blur',release);
 window.addEventListener('blur',release);window.addEventListener('pagehide',()=>disconnect());document.addEventListener('visibilitychange',()=>{if(document.hidden)disconnect();});connect.onclick=join;
 globalThis.PocketVoice={disconnect,roomChanged(room){if(room!==activeRoom&&(socket||context))disconnect();},setRoom(room,allowed){if(!allowed)disconnect();connect.disabled=!allowed;}};
})();
