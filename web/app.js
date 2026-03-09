/* ── Aurora Signal – WebRTC Demo Client ────────────────── */
const $ = (s) => document.querySelector(s);

// ── State ─────────────────────────────────────────────
let ws = null;
let selfId = null;
let roomId = null;
let displayName = null;
let localStream = null;
let intentionalLeave = false;
let reconnectAttempts = 0;
let micEnabled = true;
let camEnabled = true;

const MAX_RECONNECT = 5;
const pcs = new Map();        // peerId → RTCPeerConnection
const peerNames = new Map();  // peerId → displayName
let iceServers = [{ urls: ["stun:stun.l.google.com:19302"] }];

// ── DOM refs ──────────────────────────────────────────
const els = {
  joinPanel:  $("#joinPanel"),
  callPanel:  $("#callPanel"),
  connBadge:  $("#connBadge"),
  status:     $("#status"),
  chatLog:    $("#chatLog"),
  chatForm:   $("#chatForm"),
  chatText:   $("#chatText"),
  videoGrid:  $("#videoGrid"),
  localVideo: $("#local"),
  localLabel: $("#localLabel"),
  peerCount:  $("#peerCount"),
  peerList:   $("#peerList"),
  startBtn:   $("#start"),
  leaveBtn:   $("#leave"),
  toggleMic:  $("#toggleMic"),
  toggleCam:  $("#toggleCam"),
  micOn:      $("#micOn"),
  micOff:     $("#micOff"),
  camOn:      $("#camOn"),
  camOff:     $("#camOff"),
};

// ── Utilities ─────────────────────────────────────────
async function fetchJSON(url, opts = {}) {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

function shortId(id) {
  return id ? id.slice(0, 8) : "???";
}

function send(obj) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  ws.send(JSON.stringify({ version: "v1", ...obj }));
}

// ── Status & logging ──────────────────────────────────
function setStatus(text, type = "info") {
  els.status.textContent = text;
  els.status.className = `status status-${type}`;
}

function setBadge(state) {
  const b = els.connBadge;
  b.className = "badge";
  if (state === "online") {
    b.textContent = "在线";
    b.classList.add("badge--online");
  } else if (state === "warn") {
    b.textContent = "重连中";
    b.classList.add("badge--warn");
  } else {
    b.textContent = "离线";
  }
}

function logChat(html) {
  const div = document.createElement("div");
  div.innerHTML = `<span class="msg-time">${new Date().toLocaleTimeString()}</span>${html}`;
  els.chatLog.appendChild(div);
  els.chatLog.scrollTop = els.chatLog.scrollHeight;
}

function logSystem(text) {
  logChat(`<span class="msg-sys">${text}</span>`);
}

function logMessage(name, text) {
  const safe = text.replace(/</g, "&lt;");
  logChat(`<span class="msg-name">${name}:</span> ${safe}`);
}

// ── Participant list ──────────────────────────────────
function refreshPeerList() {
  els.peerCount.textContent = peerNames.size + 1;
  els.peerList.innerHTML = "";
  const selfLi = document.createElement("li");
  selfLi.textContent = `${displayName} (我)`;
  els.peerList.appendChild(selfLi);
  for (const [, name] of peerNames) {
    const li = document.createElement("li");
    li.textContent = name;
    els.peerList.appendChild(li);
  }
}

// ── Video cards ───────────────────────────────────────
function ensureVideoCard(peerId) {
  const existingId = `vc-${peerId}`;
  let card = document.getElementById(existingId);
  if (card) return card.querySelector("video");

  card = document.createElement("div");
  card.id = existingId;
  card.className = "video-card";

  const video = document.createElement("video");
  video.autoplay = true;
  video.playsInline = true;

  const label = document.createElement("span");
  label.className = "video-label";
  label.textContent = peerNames.get(peerId) || shortId(peerId);

  card.appendChild(video);
  card.appendChild(label);
  els.videoGrid.appendChild(card);
  return video;
}

function removeVideoCard(peerId) {
  const card = document.getElementById(`vc-${peerId}`);
  if (card) card.remove();
}

// ── RTCPeerConnection ─────────────────────────────────
function createPC(peerId) {
  const pc = new RTCPeerConnection({ iceServers });
  pcs.set(peerId, pc);

  if (localStream) {
    localStream.getTracks().forEach((t) => pc.addTrack(t, localStream));
  }

  pc.ontrack = (ev) => {
    const video = ensureVideoCard(peerId);
    video.srcObject = ev.streams[0];
  };

  pc.onicecandidate = (ev) => {
    if (ev.candidate) {
      send({ type: "trickle", to: peerId, payload: { candidate: ev.candidate } });
    }
  };

  pc.onconnectionstatechange = () => {
    const state = pc.connectionState;
    if (state === "failed") {
      logSystem(`与 ${peerNames.get(peerId) || shortId(peerId)} 的连接失败`);
    }
  };

  return pc;
}

async function negotiate(peerId) {
  const pc = pcs.get(peerId);
  if (!pc) return;
  const offer = await pc.createOffer();
  await pc.setLocalDescription(offer);
  send({ type: "offer", to: peerId, payload: { sdp: offer.sdp } });
}

function cleanupPeers() {
  for (const [, pc] of pcs) {
    try { pc.close(); } catch { /* ignore */ }
  }
  pcs.clear();
  peerNames.clear();
  // remove remote video cards
  els.videoGrid.querySelectorAll(".video-card:not(.video-card--local)").forEach((c) => c.remove());
}

// ── Room helpers ──────────────────────────────────────
async function ensureRoom(rid) {
  try {
    await fetchJSON(`/api/v1/rooms/${encodeURIComponent(rid)}`);
  } catch {
    await fetchJSON("/api/v1/rooms", {
      method: "POST",
      body: JSON.stringify({ id: rid }),
    });
  }
}

async function getToken(rid) {
  const userId = `u-${Math.random().toString(36).slice(2, 10)}`;
  const body = { userId, displayName, role: "speaker", ttlSeconds: 900 };
  const j = await fetchJSON(
    `/api/v1/rooms/${encodeURIComponent(rid)}/join-token`,
    { method: "POST", body: JSON.stringify(body) }
  );
  return j.token;
}

// ── WebSocket ─────────────────────────────────────────
async function connectWS() {
  const token = await getToken(roomId);
  const proto = location.protocol === "https:" ? "wss" : "ws";
  ws = new WebSocket(`${proto}://${location.host}/ws/v1?token=${encodeURIComponent(token)}`);

  ws.onopen = () => {
    reconnectAttempts = 0;
    setStatus("已连接，加入房间...", "info");
    send({ type: "join", payload: { roomId, displayName } });
  };

  ws.onmessage = async (e) => {
    const m = JSON.parse(e.data);
    const payload = m.payload || {};

    switch (m.type) {
      case "joined": {
        selfId = payload.self.id;
        if (payload.iceServers && payload.iceServers.length) {
          iceServers = payload.iceServers;
        }
        setBadge("online");
        setStatus(`已加入 ${roomId}`, "ok");
        for (const p of payload.peers || []) {
          peerNames.set(p.id, p.displayName || shortId(p.id));
          createPC(p.id);
          await negotiate(p.id);
        }
        refreshPeerList();
        break;
      }

      case "participant-joined": {
        const pid = payload.id;
        const name = payload.displayName || shortId(pid);
        peerNames.set(pid, name);
        createPC(pid);
        await negotiate(pid);
        refreshPeerList();
        logSystem(`${name} 加入了房间`);
        break;
      }

      case "offer": {
        const pid = m.from;
        const pc = pcs.get(pid) || createPC(pid);
        await pc.setRemoteDescription({ type: "offer", sdp: payload.sdp });
        const answer = await pc.createAnswer();
        await pc.setLocalDescription(answer);
        send({ type: "answer", to: pid, payload: { sdp: answer.sdp } });
        break;
      }

      case "answer": {
        const pc = pcs.get(m.from);
        if (pc) await pc.setRemoteDescription({ type: "answer", sdp: payload.sdp });
        break;
      }

      case "trickle": {
        const pc = pcs.get(m.from);
        if (pc) {
          try { await pc.addIceCandidate(payload.candidate); }
          catch (err) { console.warn("ICE candidate error", err); }
        }
        break;
      }

      case "participant-left": {
        const pid = payload.id;
        const name = peerNames.get(pid) || shortId(pid);
        const pc = pcs.get(pid);
        if (pc) { pc.close(); pcs.delete(pid); }
        peerNames.delete(pid);
        removeVideoCard(pid);
        refreshPeerList();
        logSystem(`${name} 离开了房间`);
        break;
      }

      case "chat": {
        const senderName = peerNames.get(m.from) || shortId(m.from);
        logMessage(senderName, payload.text || "");
        break;
      }

      case "error": {
        logSystem(`错误 [${payload.code}]: ${payload.message}`);
        setStatus(`错误: ${payload.message}`, "error");
        break;
      }

      default:
        console.log("unhandled message type", m.type, m);
    }
  };

  ws.onerror = () => setStatus("连接出错", "error");

  ws.onclose = () => {
    setBadge("offline");
    if (!intentionalLeave) {
      cleanupPeers();
      refreshPeerList();
      scheduleReconnect();
    } else {
      setStatus("已断开", "info");
    }
  };
}

async function scheduleReconnect() {
  if (intentionalLeave || reconnectAttempts >= MAX_RECONNECT) {
    if (reconnectAttempts >= MAX_RECONNECT) {
      setStatus("重连次数已用尽，请手动重新加入", "error");
    }
    return;
  }
  reconnectAttempts++;
  const delay = Math.min(1000 * 2 ** (reconnectAttempts - 1), 16000);
  setBadge("warn");
  setStatus(`连接断开，${(delay / 1000).toFixed(0)}s 后重连 (${reconnectAttempts}/${MAX_RECONNECT})...`, "warn");
  logSystem(`断线重连中... 第${reconnectAttempts}次`);
  await new Promise((r) => setTimeout(r, delay));
  if (intentionalLeave) return;
  try {
    await connectWS();
  } catch (e) {
    logSystem(`重连失败: ${e.message}`);
    scheduleReconnect();
  }
}

// ── Media controls ────────────────────────────────────
function toggleMic() {
  if (!localStream) return;
  micEnabled = !micEnabled;
  localStream.getAudioTracks().forEach((t) => (t.enabled = micEnabled));
  els.micOn.classList.toggle("hidden", !micEnabled);
  els.micOff.classList.toggle("hidden", micEnabled);
  els.toggleMic.classList.toggle("btn-icon--muted", !micEnabled);
  send({ type: micEnabled ? "unmute" : "mute", payload: { kind: "audio" } });
}

function toggleCam() {
  if (!localStream) return;
  camEnabled = !camEnabled;
  localStream.getVideoTracks().forEach((t) => (t.enabled = camEnabled));
  els.camOn.classList.toggle("hidden", !camEnabled);
  els.camOff.classList.toggle("hidden", camEnabled);
  els.toggleCam.classList.toggle("btn-icon--muted", !camEnabled);
  send({ type: camEnabled ? "unmute" : "mute", payload: { kind: "video" } });
}

// ── Join / Leave ──────────────────────────────────────
async function start() {
  roomId = $("#room").value.trim() || "room-001";
  displayName = $("#name").value.trim() || "Guest";
  intentionalLeave = false;
  reconnectAttempts = 0;
  micEnabled = true;
  camEnabled = true;

  els.startBtn.disabled = true;
  setStatus("获取媒体...", "info");

  try {
    localStream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true });
    els.localVideo.srcObject = localStream;
  } catch {
    logSystem("摄像头/麦克风获取失败，将以纯信令模式运行");
  }

  els.localLabel.textContent = displayName;

  setStatus("确认房间...", "info");
  await ensureRoom(roomId);

  setStatus("申请令牌...", "info");
  await connectWS();

  els.joinPanel.classList.add("hidden");
  els.callPanel.classList.remove("hidden");
}

function leave() {
  intentionalLeave = true;
  if (ws && ws.readyState === WebSocket.OPEN) {
    send({ type: "leave" });
    ws.close();
  }
  cleanupPeers();
  if (localStream) {
    localStream.getTracks().forEach((t) => t.stop());
    localStream = null;
  }
  els.localVideo.srcObject = null;
  els.chatLog.innerHTML = "";
  setBadge("offline");

  els.callPanel.classList.add("hidden");
  els.joinPanel.classList.remove("hidden");
  els.startBtn.disabled = false;
  setStatus("已离开房间", "info");
  refreshPeerList();
}

function sendChat() {
  const text = els.chatText.value.trim();
  if (!text || !ws) return;
  send({ type: "chat", payload: { text } });
  logMessage("我", text);
  els.chatText.value = "";
}

// ── Event binding ─────────────────────────────────────
els.startBtn.addEventListener("click", () =>
  start().catch((e) => {
    setStatus(e.message, "error");
    els.startBtn.disabled = false;
  })
);
els.leaveBtn.addEventListener("click", leave);
els.toggleMic.addEventListener("click", toggleMic);
els.toggleCam.addEventListener("click", toggleCam);
els.chatForm.addEventListener("submit", (e) => {
  e.preventDefault();
  sendChat();
});
