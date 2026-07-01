package config

import (
	"strings"
	"testing"
)

func TestLoadAllowsLoopbackDefaults(t *testing.T) {
	t.Setenv("SURGE_HOST_DOMAIN", "localhost")
	t.Setenv("SURGE_HOST_ADMIN_PASSWORD", "")
	t.Setenv("SURGE_HOST_JWT_SECRET", defaultJWTSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected loopback defaults to be allowed, got error: %v", err)
	}
	if !cfg.IsLoopbackDomain() {
		t.Fatal("expected localhost to be treated as loopback")
	}
}

func TestLoadRejectsPublicDomainWithoutPassword(t *testing.T) {
	t.Setenv("SURGE_HOST_DOMAIN", "rules.example.com")
	t.Setenv("SURGE_HOST_ADMIN_PASSWORD", "")
	t.Setenv("SURGE_HOST_JWT_SECRET", "custom-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("expected public domain without password to be rejected")
	}
	if !strings.Contains(err.Error(), "SURGE_HOST_ADMIN_PASSWORD") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsPublicDomainWithDefaultJWTSecret(t *testing.T) {
	t.Setenv("SURGE_HOST_DOMAIN", "rules.example.com")
	t.Setenv("SURGE_HOST_ADMIN_PASSWORD", "a-strong-password")
	t.Setenv("SURGE_HOST_JWT_SECRET", defaultJWTSecret)

	_, err := Load()
	if err == nil {
		t.Fatal("expected public domain with default JWT secret to be rejected")
	}
	if !strings.Contains(err.Error(), "SURGE_HOST_JWT_SECRET") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadAcceptsPublicDomainWithExplicitCredentials(t *testing.T) {
	t.Setenv("SURGE_HOST_DOMAIN", "rules.example.com")
	t.Setenv("SURGE_HOST_ADMIN_PASSWORD", "a-strong-password")
	t.Setenv("SURGE_HOST_JWT_SECRET", "custom-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected public config to load, got error: %v", err)
	}
	if cfg.IsLoopbackDomain() {
		t.Fatal("expected public domain not to be treated as loopback")
	}
	if cfg.UsesDefaultJWTSecret() {
		t.Fatal("expected custom JWT secret not to be treated as default")
	}
}
