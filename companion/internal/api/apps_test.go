package api

import "testing"

func TestResolveOIDCClientAccessPolicy(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		verifiedOnly bool
		want         string
		wantErr      bool
	}{
		{name: "approved open", status: "approved", verifiedOnly: false, want: oidcClientAccessOpen},
		{name: "approved verified only", status: "approved", verifiedOnly: true, want: oidcClientAccessVerifiedOnly},
		{name: "implicit approved", status: "", verifiedOnly: false, want: oidcClientAccessOpen},
		{name: "pending open", status: "pending", verifiedOnly: false, want: oidcClientAccessOpen},
		{name: "pending verified only", status: "pending", verifiedOnly: true, want: oidcClientAccessVerifiedOnly},
		{name: "rejected verified only", status: "rejected", verifiedOnly: true, want: oidcClientAccessVerifiedOnly},
		{name: "unknown status errors", status: "disabled", verifiedOnly: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveOIDCClientAccessPolicy(tt.status, tt.verifiedOnly)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveOIDCClientAccessPolicy(%q, %t) = %q, want %q", tt.status, tt.verifiedOnly, got, tt.want)
			}
		})
	}
}

func TestDetectLogoContentType(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantType string
		wantOK   bool
	}{
		{
			name:     "png",
			data:     []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00},
			wantType: "image/png",
			wantOK:   true,
		},
		{
			name:     "jpeg",
			data:     []byte{0xff, 0xd8, 0xff, 0xdb, 0x00, 0x43, 0x00},
			wantType: "image/jpeg",
			wantOK:   true,
		},
		{
			name:     "webp",
			data:     []byte{'R', 'I', 'F', 'F', 0x24, 0x00, 0x00, 0x00, 'W', 'E', 'B', 'P', 'V', 'P', '8', ' '},
			wantType: "image/webp",
			wantOK:   true,
		},
		{
			name:   "svg rejected",
			data:   []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotOK := detectLogoContentType(tt.data)
			if gotOK != tt.wantOK {
				t.Fatalf("detectLogoContentType() ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotType != tt.wantType {
				t.Fatalf("detectLogoContentType() type = %q, want %q", gotType, tt.wantType)
			}
		})
	}
}

func TestIsValidCallbackURL_StrictLocalhostValidation(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "https callback allowed", raw: "https://example.com/callback", want: true},
		{name: "localhost callback allowed", raw: "http://localhost:3000/callback", want: true},
		{name: "loopback callback allowed", raw: "http://127.0.0.1:3000/callback", want: true},
		{name: "localhost prefix attack rejected", raw: "http://localhost.attacker.tld/callback", want: false},
		{name: "loopback prefix attack rejected", raw: "http://127.0.0.1.attacker.tld/callback", want: false},
		{name: "userinfo rejected", raw: "https://user@example.com/callback", want: false},
		{name: "fragment rejected", raw: "https://example.com/callback#frag", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidCallbackURL(tt.raw); got != tt.want {
				t.Fatalf("isValidCallbackURL(%q) = %t, want %t", tt.raw, got, tt.want)
			}
		})
	}
}
