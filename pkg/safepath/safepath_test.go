package safepath

import "testing"

func TestNormalizeUserPath(t *testing.T) {
	tests := []struct {
		user string
		in   string
		want string
	}{
		{"asaqe", "test.yaml", "test.yaml"},
		{"asaqe", "asaqe/test.yaml", "test.yaml"},
		{"asaqe", "asaqe/subdir/test.yaml", "subdir/test.yaml"},
		{"asaqe", "other/test.yaml", "other/test.yaml"},
		{"asaqe", "subdir/asaqe/test.yaml", "subdir/asaqe/test.yaml"},
		{"asaqe", "/asaqe/test.yaml", "/asaqe/test.yaml"},
		{"asaqe", "./asaqe/test.yaml", "test.yaml"},
		{"", "asaqe/test.yaml", "asaqe/test.yaml"},
		{"asaqe", "", ""},
	}

	for _, tc := range tests {
		got := NormalizeUserPath(tc.user, tc.in)
		if got != tc.want {
			t.Fatalf("NormalizeUserPath(%q, %q) = %q, want %q", tc.user, tc.in, got, tc.want)
		}
	}
}

func TestPrepareUserPath(t *testing.T) {
	got, err := PrepareUserPath("asaqe", "  asaqe/subdir/test.yaml  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "subdir/test.yaml" {
		t.Fatalf("got %q, want subdir/test.yaml", got)
	}

	if _, err := PrepareUserPath("asaqe", "../secret.yaml"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}

	if _, err := PrepareUserPath("asaqe", ""); err == nil {
		t.Fatal("expected empty path to be rejected")
	}
}