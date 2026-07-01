package handler

import "testing"

func TestUserPathFromRequestStripsLeadingUsernamePrefix(t *testing.T) {
	got, err := userPathFromRequest("asaqe", "asaqe/subdir/test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "subdir/test.yaml" {
		t.Fatalf("got %q, want subdir/test.yaml", got)
	}
}

func TestUserPathFromRequestRejectsTraversal(t *testing.T) {
	if _, err := userPathFromRequest("asaqe", "../secret.yaml"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}
