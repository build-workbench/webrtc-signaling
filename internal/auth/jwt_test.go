package auth

import (
	"testing"
	"time"
)

func TestJWTSignAndParse(t *testing.T) {
	j := NewJWT("secret")
	tok, err := j.SignJoinToken("u1", "room-1", "speaker", time.Minute, "Alice")
	if err != nil {
		t.Fatalf("sign error: %v", err)
	}
	claims, err := j.ParseJoinToken(tok)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if claims.Subject != "u1" || claims.Rid != "room-1" || claims.Role != "speaker" || claims.DisplayName != "Alice" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}
