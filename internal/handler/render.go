package handler

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/asaqe/surge-host/internal/auth"
	"github.com/asaqe/surge-host/internal/config"
)

func loadTemplates(dir string) (*template.Template, error) {
	return template.ParseGlob(filepath.Join(dir, "*.html"))
}

func pageData(cfg *config.Config, r *http.Request, authSvc *auth.Service, extra map[string]any) map[string]any {
	exts := make([]string, 0, len(cfg.AllowedExtensions))
	for ext := range cfg.AllowedExtensions {
		exts = append(exts, ext)
	}
	// stable order for display
	for i := 0; i < len(exts); i++ {
		for j := i + 1; j < len(exts); j++ {
			if exts[j] < exts[i] {
				exts[i], exts[j] = exts[j], exts[i]
			}
		}
	}

	data := map[string]any{
		"Domain":      cfg.Domain,
		"RawBase":     rawBaseURL(cfg, r),
		"AuthEnabled": authSvc.AuthEnabled(),
		"AdminUser":   cfg.AdminUser,
		"MaxFileSize": cfg.MaxFileSize,
		"AllowedExts": strings.Join(exts, ", "),
	}
	for k, v := range extra {
		data[k] = v
	}
	return data
}

func rawBaseURL(cfg *config.Config, r *http.Request) string {
	if !cfg.IsLoopbackDomain() {
		return "https://" + cfg.Domain + "/raw"
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + "/raw"
}
