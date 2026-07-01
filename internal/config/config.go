package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

const defaultJWTSecret = "change-me-in-production"

// Config holds application configuration loaded from environment variables.
type Config struct {
	Port              int
	DataDir           string
	Domain            string
	AdminUser         string
	AdminPassword     string
	JWTSecret         string
	MaxFileSize       int64
	AllowedExtensions map[string]bool
	StaticDir         string
	TemplatesDir      string
	GitEnabled        bool
	GitAuthorName     string
	GitAuthorEmail    string
	ValidateEnabled   bool
	ValidateStrict    bool
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	port, err := envInt("SURGE_HOST_PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("invalid SURGE_HOST_PORT: %w", err)
	}

	maxSize, err := envInt64("SURGE_HOST_MAX_FILE_SIZE", 5*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("invalid SURGE_HOST_MAX_FILE_SIZE: %w", err)
	}

	exts := envString("SURGE_HOST_ALLOWED_EXTENSIONS", ".conf,.list,.txt,.module,.yaml,.yml,.json")
	allowed := make(map[string]bool)
	for _, ext := range strings.Split(exts, ",") {
		ext = strings.TrimSpace(strings.ToLower(ext))
		if ext != "" {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			allowed[ext] = true
		}
	}

	gitEnabled, err := envBool("SURGE_HOST_GIT_ENABLED", true)
	if err != nil {
		return nil, fmt.Errorf("invalid SURGE_HOST_GIT_ENABLED: %w", err)
	}

	validateEnabled, err := envBool("SURGE_HOST_VALIDATE_ENABLED", true)
	if err != nil {
		return nil, fmt.Errorf("invalid SURGE_HOST_VALIDATE_ENABLED: %w", err)
	}
	validateStrict, err := envBool("SURGE_HOST_VALIDATE_STRICT", false)
	if err != nil {
		return nil, fmt.Errorf("invalid SURGE_HOST_VALIDATE_STRICT: %w", err)
	}

	cfg := &Config{
		Port:              port,
		DataDir:           envString("SURGE_HOST_DATA_DIR", "./data"),
		Domain:            envString("SURGE_HOST_DOMAIN", "localhost"),
		AdminUser:         envString("SURGE_HOST_ADMIN_USER", "admin"),
		AdminPassword:     envString("SURGE_HOST_ADMIN_PASSWORD", ""),
		JWTSecret:         envString("SURGE_HOST_JWT_SECRET", defaultJWTSecret),
		MaxFileSize:       maxSize,
		AllowedExtensions: allowed,
		StaticDir:         envString("SURGE_HOST_STATIC_DIR", "./web/static"),
		TemplatesDir:      envString("SURGE_HOST_TEMPLATES_DIR", "./web/templates"),
		GitEnabled:        gitEnabled,
		GitAuthorName:     envString("SURGE_HOST_GIT_AUTHOR_NAME", "surge-host"),
		GitAuthorEmail:    envString("SURGE_HOST_GIT_AUTHOR_EMAIL", "surge-host@localhost"),
		ValidateEnabled:   validateEnabled,
		ValidateStrict:    validateStrict,
	}

	if err := validateSecurityBaseline(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// DBPath returns the SQLite database file path.
func (c *Config) DBPath() string {
	return c.DataDir + "/surge-host.db"
}

// UsersDir returns the base directory for user file storage.
func (c *Config) UsersDir() string {
	return c.DataDir + "/users"
}

// UserDir returns the storage directory for a specific user.
func (c *Config) UserDir(username string) string {
	return c.UsersDir() + "/" + username
}

// RawURL builds the public raw URL for a user file.
func (c *Config) RawURL(username, filename string) string {
	scheme := "https"
	if c.IsLoopbackDomain() {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/raw/%s/%s", scheme, c.Domain, username, filename)
}

// IsLoopbackDomain reports whether the configured domain is intended for local-only development.
func (c *Config) IsLoopbackDomain() bool {
	host := normalizeDomainHost(c.Domain)
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// UsesDefaultJWTSecret reports whether the configured JWT secret is still the unsafe fallback value.
func (c *Config) UsesDefaultJWTSecret() bool {
	return c.JWTSecret == defaultJWTSecret
}

func validateSecurityBaseline(cfg *Config) error {
	if cfg.IsLoopbackDomain() {
		return nil
	}
	if strings.TrimSpace(cfg.AdminPassword) == "" {
		return fmt.Errorf("SURGE_HOST_ADMIN_PASSWORD is required when SURGE_HOST_DOMAIN is not loopback")
	}
	if cfg.UsesDefaultJWTSecret() {
		return fmt.Errorf("SURGE_HOST_JWT_SECRET must be changed when SURGE_HOST_DOMAIN is not loopback")
	}
	return nil
}

func normalizeDomainHost(domain string) string {
	host := strings.TrimSpace(strings.ToLower(domain))
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	} else if strings.Count(host, ":") == 1 {
		if rawHost, _, ok := strings.Cut(host, ":"); ok {
			host = rawHost
		}
	}
	return strings.Trim(host, "[]")
}

func envString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func envInt64(key string, fallback int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func envBool(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean: %s", v)
	}
}
