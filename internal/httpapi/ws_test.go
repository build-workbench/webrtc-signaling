package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"signal/internal/auth"
	"signal/internal/config"
	"signal/internal/room"
	"signal/internal/signaling"
)

func testServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	cfg := &config.Config{
		LogLevel: "error",
		Server: config.ServerCfg{
			Addr:            ":0",
			MaxMsgBytes:     65536,
			PingIntervalSec: 30,
			PongWaitSec:     60,
		},
		Security: config.SecurityCfg{
			JWTSecret: "test-secret-for-integration",
			RateLimit: config.RateLimitCfg{WSPerConnRPS: 100, WSBurst: 200},
		},
		Turn: config.TurnCfg{
			STUN: []string{"stun:stun.l.google.com:19302"},
		},
	}
	log, _ := zap.NewDevelopment()
	mgr := room.NewManager(log)
	jwtAuth := auth.NewJWT(cfg.Security.JWTSecret)
	srv := NewServer(cfg, log, mgr, jwtAuth)
	ts := httptest.NewServer(srv.httpSrv.Handler)
	return srv, ts
}

func wsConnect(t *testing.T, ts *httptest.Server, token string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/v1?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	return conn
}

func getToken(t *testing.T, ts *httptest.Server, roomID, userID, name string) string {
	t.Helper()
	body := `{"userId":"` + userID + `","displayName":"` + name + `","role":"speaker","ttlSeconds":60}`
	resp, err := http.Post(ts.URL+"/api/v1/rooms/"+roomID+"/join-token", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("get token status: %d", resp.StatusCode)
	}
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return result["token"].(string)
}

func readEnvelope(t *testing.T, conn *websocket.Conn) signaling.Envelope {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var env signaling.Envelope
	if err := conn.ReadJSON(&env); err != nil {
		t.Fatalf("readJSON: %v", err)
	}
	return env
}

func TestWSJoinAndLeave(t *testing.T) {
	_, ts := testServer(t)
	defer ts.Close()

	tok := getToken(t, ts, "room-int", "u1", "Alice")
	conn := wsConnect(t, ts, tok)
	defer conn.Close()

	// send join
	_ = conn.WriteJSON(signaling.Envelope{Type: signaling.TypeJoin, Payload: mustJSON(signaling.JoinPayload{RoomID: "room-int", DisplayName: "Alice"})})

	// expect joined
	env := readEnvelope(t, conn)
	if env.Type != signaling.TypeJoined {
		t.Fatalf("expected joined, got %s", env.Type)
	}
	var payload map[string]any
	_ = json.Unmarshal(env.Payload, &payload)
	self := payload["self"].(map[string]any)
	if self["displayName"] != "Alice" {
		t.Fatalf("expected displayName Alice, got %v", self["displayName"])
	}

	// send leave
	_ = conn.WriteJSON(signaling.Envelope{Type: signaling.TypeLeave})
}

func TestWSTwoPeersSignaling(t *testing.T) {
	_, ts := testServer(t)
	defer ts.Close()

	tokA := getToken(t, ts, "room-2p", "u1", "Alice")
	tokB := getToken(t, ts, "room-2p", "u2", "Bob")

	connA := wsConnect(t, ts, tokA)
	defer connA.Close()

	// Alice joins
	_ = connA.WriteJSON(signaling.Envelope{Type: signaling.TypeJoin, Payload: mustJSON(signaling.JoinPayload{RoomID: "room-2p", DisplayName: "Alice"})})
	envA := readEnvelope(t, connA)
	if envA.Type != signaling.TypeJoined {
		t.Fatalf("A: expected joined, got %s", envA.Type)
	}
	var payloadA map[string]any
	_ = json.Unmarshal(envA.Payload, &payloadA)
	peers := payloadA["peers"].([]any)
	if len(peers) != 0 {
		t.Fatalf("A: expected 0 peers, got %d", len(peers))
	}

	// Bob joins
	connB := wsConnect(t, ts, tokB)
	defer connB.Close()
	_ = connB.WriteJSON(signaling.Envelope{Type: signaling.TypeJoin, Payload: mustJSON(signaling.JoinPayload{RoomID: "room-2p", DisplayName: "Bob"})})
	envB := readEnvelope(t, connB)
	if envB.Type != signaling.TypeJoined {
		t.Fatalf("B: expected joined, got %s", envB.Type)
	}
	var payloadB map[string]any
	_ = json.Unmarshal(envB.Payload, &payloadB)
	peersB := payloadB["peers"].([]any)
	if len(peersB) != 1 {
		t.Fatalf("B: expected 1 peer, got %d", len(peersB))
	}

	// Alice receives participant-joined
	envPJ := readEnvelope(t, connA)
	if envPJ.Type != signaling.TypePeerJoin {
		t.Fatalf("A: expected participant-joined, got %s", envPJ.Type)
	}
	bobPeerID := envB.From

	// Alice -> Bob: offer
	_ = connA.WriteJSON(signaling.Envelope{Type: signaling.TypeOffer, To: bobPeerID, Payload: mustJSON(map[string]any{"sdp": "v=0 offer"})})
	envOffer := readEnvelope(t, connB)
	if envOffer.Type != signaling.TypeOffer {
		t.Fatalf("B: expected offer, got %s", envOffer.Type)
	}

	alicePeerID := envA.From

	// Bob -> Alice: answer
	_ = connB.WriteJSON(signaling.Envelope{Type: signaling.TypeAnswer, To: alicePeerID, Payload: mustJSON(map[string]any{"sdp": "v=0 answer"})})
	envAnswer := readEnvelope(t, connA)
	if envAnswer.Type != signaling.TypeAnswer {
		t.Fatalf("A: expected answer, got %s", envAnswer.Type)
	}

	// Alice -> Bob: trickle
	_ = connA.WriteJSON(signaling.Envelope{Type: signaling.TypeTrickle, To: bobPeerID, Payload: mustJSON(map[string]any{"candidate": "candidate:1"})})
	envTrickle := readEnvelope(t, connB)
	if envTrickle.Type != signaling.TypeTrickle {
		t.Fatalf("B: expected trickle, got %s", envTrickle.Type)
	}

	// verify ts and id are populated
	if envTrickle.Ts == 0 {
		t.Fatal("expected ts to be populated")
	}
	if envTrickle.ID == "" {
		t.Fatal("expected id to be populated")
	}

	// Bob leaves, Alice should receive participant-left
	_ = connB.WriteJSON(signaling.Envelope{Type: signaling.TypeLeave})
	envLeft := readEnvelope(t, connA)
	if envLeft.Type != signaling.TypePeerLeave {
		t.Fatalf("A: expected participant-left, got %s", envLeft.Type)
	}
}

func TestWSChat(t *testing.T) {
	_, ts := testServer(t)
	defer ts.Close()

	tokA := getToken(t, ts, "room-chat", "u1", "Alice")
	tokB := getToken(t, ts, "room-chat", "u2", "Bob")

	connA := wsConnect(t, ts, tokA)
	defer connA.Close()
	_ = connA.WriteJSON(signaling.Envelope{Type: signaling.TypeJoin, Payload: mustJSON(signaling.JoinPayload{RoomID: "room-chat"})})
	readEnvelope(t, connA) // joined

	connB := wsConnect(t, ts, tokB)
	defer connB.Close()
	_ = connB.WriteJSON(signaling.Envelope{Type: signaling.TypeJoin, Payload: mustJSON(signaling.JoinPayload{RoomID: "room-chat"})})
	readEnvelope(t, connB) // joined
	readEnvelope(t, connA) // participant-joined

	// broadcast chat
	_ = connA.WriteJSON(signaling.Envelope{Type: signaling.TypeChat, Payload: mustJSON(map[string]any{"text": "hello"})})
	envChat := readEnvelope(t, connB)
	if envChat.Type != signaling.TypeChat {
		t.Fatalf("B: expected chat, got %s", envChat.Type)
	}
	var chatPayload map[string]any
	_ = json.Unmarshal(envChat.Payload, &chatPayload)
	if chatPayload["text"] != "hello" {
		t.Fatalf("expected text 'hello', got %v", chatPayload["text"])
	}
}

func TestWSErrorCases(t *testing.T) {
	_, ts := testServer(t)
	defer ts.Close()

	t.Run("missing_token", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws/v1"
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			t.Fatal("expected error")
		}
		if resp != nil && resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("first_msg_not_join", func(t *testing.T) {
		tok := getToken(t, ts, "room-err", "u1", "A")
		conn := wsConnect(t, ts, tok)
		defer conn.Close()
		_ = conn.WriteJSON(signaling.Envelope{Type: signaling.TypeChat, Payload: mustJSON(map[string]any{"text": "hi"})})
		env := readEnvelope(t, conn)
		if env.Type != signaling.TypeError {
			t.Fatalf("expected error, got %s", env.Type)
		}
	})

	t.Run("offer_missing_to", func(t *testing.T) {
		tok := getToken(t, ts, "room-err2", "u1", "A")
		conn := wsConnect(t, ts, tok)
		defer conn.Close()
		_ = conn.WriteJSON(signaling.Envelope{Type: signaling.TypeJoin, Payload: mustJSON(signaling.JoinPayload{RoomID: "room-err2"})})
		readEnvelope(t, conn) // joined
		_ = conn.WriteJSON(signaling.Envelope{Type: signaling.TypeOffer, Payload: mustJSON(map[string]any{"sdp": "v=0"})})
		env := readEnvelope(t, conn)
		if env.Type != signaling.TypeError {
			t.Fatalf("expected error, got %s", env.Type)
		}
	})

	t.Run("unsupported_type", func(t *testing.T) {
		tok := getToken(t, ts, "room-err3", "u1", "A")
		conn := wsConnect(t, ts, tok)
		defer conn.Close()
		_ = conn.WriteJSON(signaling.Envelope{Type: signaling.TypeJoin, Payload: mustJSON(signaling.JoinPayload{RoomID: "room-err3"})})
		readEnvelope(t, conn) // joined
		_ = conn.WriteJSON(signaling.Envelope{Type: "unknown_type"})
		env := readEnvelope(t, conn)
		if env.Type != signaling.TypeError {
			t.Fatalf("expected error, got %s", env.Type)
		}
	})
}

func TestRESTEndpoints(t *testing.T) {
	_, ts := testServer(t)
	defer ts.Close()

	// healthz
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("healthz: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// readyz
	resp, err = http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("readyz: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// create room
	resp, err = http.Post(ts.URL+"/api/v1/rooms", "application/json", strings.NewReader(`{"id":"test-room","maxParticipants":4}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("create room: %d", resp.StatusCode)
	}
	var roomResp map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&roomResp)
	resp.Body.Close()
	if roomResp["id"] != "test-room" {
		t.Fatalf("expected id test-room, got %v", roomResp["id"])
	}

	// get room
	resp, err = http.Get(ts.URL + "/api/v1/rooms/test-room")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("get room: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// room not found
	resp, err = http.Get(ts.URL + "/api/v1/rooms/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// ice servers
	resp, err = http.Get(ts.URL + "/api/v1/ice-servers")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("ice-servers: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// security headers check
	resp, err = http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing X-Content-Type-Options header")
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID header")
	}
	resp.Body.Close()
}
