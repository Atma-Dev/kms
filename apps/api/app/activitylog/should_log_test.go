package activitylog

import "testing"

func TestShouldLog(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{"GET", "/api/deposits", false},
		{"GET", "/api/management/audits", false},
		{"POST", "/api/deposits", true},
		{"PATCH", "/api/management/users/1", true},
		{"PUT", "/api/partners/1", true},
		{"DELETE", "/api/partners/1", true},
		{"POST", "/api/auth/logout", false},
		{"PATCH", "/api/profiles/me", false},
		{"PATCH", "/api/profiles/toggle-user-status", true},
	}
	for _, tc := range cases {
		if got := ShouldLog(tc.method, tc.path); got != tc.want {
			t.Errorf("ShouldLog(%q, %q) = %v, want %v", tc.method, tc.path, got, tc.want)
		}
	}
}
