import http from 'k6/http';
import { check, sleep } from 'k6';
import ws from 'k6/ws';

export let options = {
  vus: 1,
  duration: '10s',
};

const base = __ENV.BASE_URL || 'http://localhost:8080';
const roomId = __ENV.ROOM_ID || 'room-001';

export default function () {
  // create room (idempotent)
  http.post(`${base}/api/v1/rooms`, JSON.stringify({ id: roomId }), { headers: { 'Content-Type': 'application/json' } });
  // get token
  const tokRes = http.post(`${base}/api/v1/rooms/${roomId}/join-token`, JSON.stringify({ userId: `k6-${__ITER}`, displayName: 'k6', role: 'speaker', ttlSeconds: 60 }), { headers: { 'Content-Type': 'application/json' } });
  check(tokRes, { 'token ok': (r) => r.status === 200 });
  const tok = tokRes.json('token');

  const url = `${base.replace('http', 'ws')}/ws/v1?token=${encodeURIComponent(tok)}`;
  const params = { tags: { my_tag: 'websocket' } };

  const res = ws.connect(url, params, function (socket) {
    socket.on('open', function () {
      socket.send(JSON.stringify({ version: 'v1', type: 'join', payload: { roomId, displayName: 'k6' } }));
    });

    socket.on('message', function (data) {
      const m = JSON.parse(data);
      if (m.type === 'joined') {
        socket.send(JSON.stringify({ version: 'v1', type: 'chat', payload: { text: 'hello' } }));
      }
    });

    socket.setTimeout(function () { socket.close(); }, 5000);
  });

  check(res, { 'ws status 101': (r) => r && r.status === 101 });
  sleep(1);
}
