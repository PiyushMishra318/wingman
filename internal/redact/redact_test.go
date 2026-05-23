package redact

import (
	"strings"
	"testing"
)

func TestString_redactsProfilePaths(t *testing.T) {
	t.Setenv("USERPROFILE", `C:\Users\TestUser`)
	t.Setenv("USERNAME", "TestUser")

	in := `failed at C:\Users\TestUser\AppData\Local\winget\logs`
	got := String(in)
	if strings.Contains(got, "TestUser") {
		t.Fatalf("expected redacted output, got %q", got)
	}
	if !strings.Contains(got, "<userprofile>") && !strings.Contains(got, "<user>") {
		t.Fatalf("expected placeholder, got %q", got)
	}
}

func TestString_empty(t *testing.T) {
	if String("") != "" {
		t.Fatal("expected empty")
	}
}
