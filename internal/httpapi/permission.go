package httpapi

import "strings"

// Role represents a participant's role in a room.
type Role string

const (
	RoleViewer    Role = "viewer"
	RoleSpeaker   Role = "speaker"
	RoleModerator Role = "moderator"
)

// Policy defines permission checking for signaling messages.
type Policy interface {
	Normalize(raw string) Role
	CanSignalMedia(role Role) bool
	CanModerateOthers(role Role) bool
}

// DefaultPolicy is the standard three-tier permission policy.
type DefaultPolicy struct{}

func NewPolicy() *DefaultPolicy { return &DefaultPolicy{} }

// Normalize converts a raw role string to a Role.
// Empty string defaults to speaker; unknown roles return empty string.
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

// CanSignalMedia returns true if the role can send offer/answer/trickle.
func (p *DefaultPolicy) CanSignalMedia(role Role) bool { return role != RoleViewer }

// CanModerateOthers returns true if the role can mute/unmute other participants.
func (p *DefaultPolicy) CanModerateOthers(role Role) bool { return role == RoleModerator }
