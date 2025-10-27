const $ = (s)=>document.querySelector(s);
let ws;
let selfId;
let roomId;
let displayName;
let pcs = new Map();
let localStream;

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

function setStatus(t){ $('#status').textContent = t; }

async function getToken(rid, name){
  const userId = `u-${Math.random().toString(36).slice(2,8)}`;
  const body = {userId, displayName: name, role:'speaker', ttlSeconds: 900};
  const j = await fetchJSON(`/api/v1/rooms/${encodeURIComponent(rid)}/join-token`, {
    method:'POST', body: JSON.stringify(body)
  });
  return j.token;
}

async function getICEServers(){
  try {
    const j = await fetchJSON('/api/v1/ice-servers');
    return j;
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

async function createPC(peerId, polite=false){
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
  return pc;
}

function send(obj){
  ws.send(JSON.stringify(Object.assign({version:'v1'}, obj)));
}

async function start(){
  roomId = $('#room').value.trim() || 'room-001';
  displayName = $('#name').value.trim() || 'Guest';
  setStatus('获取媒体...');
  try{
    localStream = await navigator.mediaDevices.getUserMedia({audio:true, video:true});
    $('#local').srcObject = localStream;
  }catch(e){
    console.warn('getUserMedia failed', e);
  }
  setStatus('申请令牌...');
  const token = await getToken(roomId, displayName);
  setStatus('连接信令...');
  ws = new WebSocket(`${location.protocol==='https:'?'wss':'ws'}://${location.host}/ws/v1?token=${encodeURIComponent(token)}`);
  ws.onopen = ()=>{
    setStatus('已连接，加入房间...');
    send({type:'join', payload:{roomId, displayName}});
  };
  ws.onmessage = async (e)=>{
    const m = JSON.parse(e.data);
    if(m.type==='joined'){
      selfId = m.payload.self.id;
      setStatus(`加入成功，ID=${selfId}，房间=${roomId}`);
      for(const p of m.payload.peers){
        const pc = await createPC(p.id);
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        send({type:'offer', to:p.id, payload:{sdp: offer.sdp}});
      }
    }else if(m.type==='participant-joined'){
      const pid = m.payload.id;
      const pc = await createPC(pid);
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      send({type:'offer', to:pid, payload:{sdp: offer.sdp}});
      log(`peer ${pid} joined`);
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
      log(`peer ${pid} left`);
    }else if(m.type==='chat'){
      log(`${m.from||'system'}: ${m.payload.text}`);
    }else if(m.type==='error'){
      log(`error: ${m.payload.code} ${m.payload.message}`);
    }
  };
  ws.onclose = ()=> setStatus('连接关闭');
  $('#start').disabled = true;
  $('#leave').disabled = false;
}

async function leave(){
  if(ws){ send({type:'leave'}); ws.close(); }
  for(const [id, pc] of pcs){ try{ pc.close(); }catch{} }
  pcs.clear();
  $('#start').disabled = false;
  $('#leave').disabled = true;
}

async function sendChat(){
  const t = $('#chatText').value.trim();
  if(!t || !ws) return;
  send({type:'chat', payload:{text:t}});
  $('#chatText').value='';
}

$('#start').addEventListener('click', ()=> start().catch(e=> setStatus(e.message)));
$('#leave').addEventListener('click', ()=> leave());
$('#sendChat').addEventListener('click', ()=> sendChat());
