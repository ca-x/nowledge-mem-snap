package config

import "testing"

func TestNormalizeBasePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "root", in: "/", want: ""},
		{name: "adds leading slash", in: "admin", want: "/admin"},
		{name: "trims trailing slash", in: "/admin/", want: "/admin"},
		{name: "keeps nested path", in: " tools/app ", want: "/tools/app"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeBasePath(tt.in); got != tt.want {
				t.Fatalf("NormalizeBasePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateBasePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{name: "empty", in: "", wantErr: false},
		{name: "valid", in: "/admin", wantErr: false},
		{name: "valid nested", in: "/tools/app", wantErr: false},
		{name: "missing leading slash", in: "admin", wantErr: true},
		{name: "trailing slash", in: "/admin/", wantErr: true},
		{name: "query", in: "/admin?x=1", wantErr: true},
		{name: "fragment", in: "/admin#x", wantErr: true},
		{name: "empty segment", in: "/admin//app", wantErr: true},
		{name: "dot segment", in: "/admin/./app", wantErr: true},
		{name: "parent segment", in: "/admin/../app", wantErr: true},
		{name: "space", in: "/admin snap", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateBasePath(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateBasePath(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
		})
	}
}
