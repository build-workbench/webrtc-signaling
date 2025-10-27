package signaling

import "encoding/json"

type MessageType string

const (
	TypeJoin      MessageType = "join"
	TypeJoined    MessageType = "joined"
	TypeOffer     MessageType = "offer"
	TypeAnswer    MessageType = "answer"
	TypeTrickle   MessageType = "trickle"
	TypeLeave     MessageType = "leave"
	TypeChat      MessageType = "chat"
	TypeMute      MessageType = "mute"
	TypeUnmute    MessageType = "unmute"
	TypeError     MessageType = "error"
	TypePeerJoin  MessageType = "participant-joined"
	TypePeerLeave MessageType = "participant-left"
)

type Envelope struct {
	ID      string          `json:"id,omitempty"`
	Version string          `json:"version,omitempty"`
	Type    MessageType     `json:"type"`
	RoomID  string          `json:"roomId,omitempty"`
	From    string          `json:"from,omitempty"`
	To      string          `json:"to,omitempty"`
	Ts      int64           `json:"ts,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type JoinPayload struct {
	RoomID      string `json:"roomId"`
	DisplayName string `json:"displayName,omitempty"`
	Role        string `json:"role,omitempty"`
}

type SDP struct {
	SDP string `json:"sdp"`
}

type Trickle struct {
	Candidate any `json:"candidate"`
}

type Chat struct {
	To   string `json:"to,omitempty"`
	Text string `json:"text"`
}

type ErrorPayload struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}
