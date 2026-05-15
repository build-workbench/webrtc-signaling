package permission

import "testing"

func TestNormalize(t *testing.T) {
	p := NewPolicy()
	tests := []struct {
		input    string
		expected Role
	}{
		{"", RoleSpeaker},
		{"speaker", RoleSpeaker},
		{"SPEAKER", RoleSpeaker},
		{"  speaker  ", RoleSpeaker},
		{"viewer", RoleViewer},
		{"VIEWER", RoleViewer},
		{"moderator", RoleModerator},
		{"Moderator", RoleModerator},
		{"invalid", ""},
		{"admin", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := p.Normalize(tt.input)
			if got != tt.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCanSignalMedia(t *testing.T) {
	p := NewPolicy()
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleViewer, false},
		{RoleSpeaker, true},
		{RoleModerator, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := p.CanSignalMedia(tt.role)
			if got != tt.expected {
				t.Errorf("CanSignalMedia(%q) = %v, want %v", tt.role, got, tt.expected)
			}
		})
	}
}

func TestCanModerateOthers(t *testing.T) {
	p := NewPolicy()
	tests := []struct {
		role     Role
		expected bool
	}{
		{RoleViewer, false},
		{RoleSpeaker, false},
		{RoleModerator, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			got := p.CanModerateOthers(tt.role)
			if got != tt.expected {
				t.Errorf("CanModerateOthers(%q) = %v, want %v", tt.role, got, tt.expected)
			}
		})
	}
}
