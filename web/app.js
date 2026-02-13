const $ = (s)=>document.querySelector(s);
let ws;
let selfId;
let roomId;
let displayName;
let pcs = new Map();
let localStream;
let intentionalLeave = false;
let reconnectAttempts = 0;
const MAX_RECONNECT = 5;
let cachedICE = null;

async function fetchJSON(url, opts={}){
  const res = await fetch(url, Object.assign({headers:{'Content-Type':'application/json'}}, opts));
  if(!res.ok) throw new Error(`HTTP ${res.status}`);
  return await res.json();
}

function log(msg){
  const el = $('#chatLog');
  el.textContent += `\n${new Date().toLocaleTimeString()} ${msg}`;
  el.scrollTop = el.scrollHeight;
}

function setStatus(t, type='info'){
  const el = $('#status');
  el.textContent = t;
  el.className = `status status-${type}`;
}

async function getToken(rid, name){
  const userId = `u-${Math.random().toString(36).slice(2,8)}`;
  const body = {userId, displayName: name, role:'speaker', ttlSeconds: 900};
  const j = await fetchJSON(`/api/v1/rooms/${encodeURIComponent(rid)}/join-token`, {
    method:'POST', body: JSON.stringify(body)
  });
  return j.token;
}

async function getICEServers(){
  if(cachedICE) return cachedICE;
  try {
    cachedICE = await fetchJSON('/api/v1/ice-servers');
    return cachedICE;
  } catch(e) {
    return [{urls:['stun:stun.l.google.com:19302']}];
  }
}

function ensureVideo(id){
  let v = document.getElementById(`v-${id}`);
  if(!v){
    v = document.createElement('video');
    v.id = `v-${id}`;
    v.autoplay = true;
    v.playsInline = true;
    $('#remotes').appendChild(v);
  }
  return v;
}

function removeVideo(id){
  const v = document.getElementById(`v-${id}`);
  if(v && v.parentNode) v.parentNode.removeChild(v);
}

async function createPC(peerId){
  const iceServers = await getICEServers();
  const pc = new RTCPeerConnection({iceServers});
  pcs.set(peerId, pc);
  if(localStream){
    localStream.getTracks().forEach(t => pc.addTrack(t, localStream));
  }
  pc.ontrack = (ev)=>{
    const v = ensureVideo(peerId);
    v.srcObject = ev.streams[0];
  };
  pc.onicecandidate = (ev)=>{
    if(ev.candidate){
      send({type:'trickle', to: peerId, payload:{candidate: ev.candidate}});
    }
  };
  pc.onconnectionstatechange = ()=>{
    if(pc.connectionState === 'failed'){
      log(`与 ${peerId.slice(0,8)} 的连接失败`);
    }
  };
  return pc;
}

function send(obj){
  if(!ws || ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify(Object.assign({version:'v1'}, obj)));
}

function cleanupPeers(){
  for(const [id, pc] of pcs){ try{ pc.close(); }catch{} }
  pcs.clear();
  $('#remotes').innerHTML = '';
}

async function scheduleReconnect(){
  if(intentionalLeave || reconnectAttempts >= MAX_RECONNECT) return;
  reconnectAttempts++;
  const delay = Math.min(1000 * Math.pow(2, reconnectAttempts - 1), 16000);
  setStatus(`连接断开，${(delay/1000).toFixed(0)}s 后重连 (${reconnectAttempts}/${MAX_RECONNECT})...`, 'warn');
  log(`断线重连中... 第${reconnectAttempts}次`);
  await new Promise(r => setTimeout(r, delay));
  if(intentionalLeave) return;
  try {
    await connectWS();
  } catch(e) {
    log(`重连失败: ${e.message}`);
    scheduleReconnect();
  }
}

async function connectWS(){
  const token = await getToken(roomId, displayName);
  ws = new WebSocket(`${location.protocol==='https:'?'wss':'ws'}://${location.host}/ws/v1?token=${encodeURIComponent(token)}`);
  ws.onopen = ()=>{
    reconnectAttempts = 0;
    setStatus('已连接，加入房间...', 'info');
    send({type:'join', payload:{roomId, displayName}});
  };
  ws.onmessage = async (e)=>{
    const m = JSON.parse(e.data);
    if(m.type==='joined'){
      selfId = m.payload.self.id;
      setStatus(`已加入 ${roomId}`, 'ok');
      for(const p of m.payload.peers){
        const pc = await createPC(p.id);
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        send({type:'offer', to:p.id, payload:{sdp: offer.sdp}});
      }
    }else if(m.type==='participant-joined'){
      const pid = m.payload.id;
      const name = m.payload.displayName || pid.slice(0,8);
      const pc = await createPC(pid);
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      send({type:'offer', to:pid, payload:{sdp: offer.sdp}});
      log(`${name} 加入了房间`);
    }else if(m.type==='offer'){
      const pid = m.from;
      const pc = pcs.get(pid) || await createPC(pid);
      await pc.setRemoteDescription({type:'offer', sdp:m.payload.sdp});
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      send({type:'answer', to: pid, payload:{sdp: answer.sdp}});
    }else if(m.type==='answer'){
      const pid = m.from;
      const pc = pcs.get(pid);
      if(pc){ await pc.setRemoteDescription({type:'answer', sdp:m.payload.sdp}); }
    }else if(m.type==='trickle'){
      const pid = m.from;
      const pc = pcs.get(pid);
      if(pc){ try { await pc.addIceCandidate(m.payload.candidate); } catch(e) { console.warn(e); } }
    }else if(m.type==='participant-left'){
      const pid = m.payload.id;
      const pc = pcs.get(pid);
      if(pc){ pc.close(); pcs.delete(pid); }
      removeVideo(pid);
      log(`${pid.slice(0,8)} 离开了房间`);
    }else if(m.type==='chat'){
      log(`${m.from ? m.from.slice(0,8) : '系统'}: ${m.payload.text}`);
    }else if(m.type==='error'){
      log(`错误 [${m.payload.code}]: ${m.payload.message}`);
      setStatus(`错误: ${m.payload.message}`, 'error');
    }
  };
  ws.onerror = ()=> setStatus('连接出错', 'error');
  ws.onclose = ()=>{
    if(!intentionalLeave){
      cleanupPeers();
      scheduleReconnect();
    } else {
      setStatus('已断开', 'info');
    }
  };
}

async function start(){
  roomId = $('#room').value.trim() || 'room-001';
  displayName = $('#name').value.trim() || 'Guest';
  intentionalLeave = false;
  reconnectAttempts = 0;
  setStatus('获取媒体...', 'info');
  try{
    localStream = await navigator.mediaDevices.getUserMedia({audio:true, video:true});
    $('#local').srcObject = localStream;
  }catch(e){
    log('摄像头/麦克风获取失败，将以纯信令模式运行');
  }
  setStatus('申请令牌...', 'info');
  await connectWS();
  $('#start').disabled = true;
  $('#leave').disabled = false;
}

async function leave(){
  intentionalLeave = true;
  if(ws && ws.readyState === WebSocket.OPEN){ send({type:'leave'}); ws.close(); }
  cleanupPeers();
  if(localStream){ localStream.getTracks().forEach(t => t.stop()); localStream = null; }
  cachedICE = null;
  $('#local').srcObject = null;
  $('#start').disabled = false;
  $('#leave').disabled = true;
  setStatus('已离开房间', 'info');
}

async function sendChat(){
  const t = $('#chatText').value.trim();
  if(!t || !ws) return;
  send({type:'chat', payload:{text:t}});
  log(`我: ${t}`);
  $('#chatText').value='';
}

$('#start').addEventListener('click', ()=> start().catch(e=> setStatus(e.message, 'error')));
$('#leave').addEventListener('click', ()=> leave());
$('#sendChat').addEventListener('click', ()=> sendChat());
$('#chatText').addEventListener('keydown', (e)=>{ if(e.key==='Enter') sendChat(); });
