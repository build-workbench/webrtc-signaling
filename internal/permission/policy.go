package permission

import "strings"

// Role represents a participant's role in a room.
type Role string

const (
	RoleViewer    Role = "viewer"
	RoleSpeaker   Role = "speaker"
	RoleModerator Role = "moderator"
)

// Policy defines the interface for permission checking.
type Policy interface {
	// Normalize converts a raw role string to a Role.
	// Returns empty string if the role is invalid.
	Normalize(raw string) Role

	// CanSignalMedia returns true if the role can send offer/answer/trickle messages.
	CanSignalMedia(role Role) bool

	// CanModerateOthers returns true if the role can mute/unmute other participants.
	CanModerateOthers(role Role) bool
}

// DefaultPolicy is the standard permission policy for the signaling server.
type DefaultPolicy struct{}

// NewPolicy creates a new DefaultPolicy.
func NewPolicy() *DefaultPolicy {
	return &DefaultPolicy{}
}

// Normalize converts a raw role string to a Role.
func (p *DefaultPolicy) Normalize(raw string) Role {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "speaker":
		return RoleSpeaker
	case "viewer":
		return RoleViewer
	case "moderator":
		return RoleModerator
	default:
		return ""
	}
}

// CanSignalMedia returns true if the role can send offer/answer/trickle messages.
// Viewers cannot signal media; speakers and moderators can.
func (p *DefaultPolicy) CanSignalMedia(role Role) bool {
	return role != RoleViewer
}

// CanModerateOthers returns true if the role can mute/unmute other participants.
// Only moderators can moderate others.
func (p *DefaultPolicy) CanModerateOthers(role Role) bool {
	return role == RoleModerator
}
