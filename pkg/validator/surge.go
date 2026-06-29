package validator

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

// Level indicates issue severity.
const (
	LevelError   = "error"
	LevelWarning = "warning"
)

// Issue describes a single validation problem.
type Issue struct {
	Line    int    `json:"line"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
	Level   string `json:"level"`
}

// Result holds validation output.
type Result struct {
	Valid    bool    `json:"valid"`
	FileType string  `json:"file_type"`
	Issues   []Issue `json:"issues"`
}

// ValidationError is returned when strict validation fails on save.
type ValidationError struct {
	Result Result
}

func (e *ValidationError) Error() string {
	if len(e.Result.Issues) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s", e.Result.Issues[0].Message)
}

var ruleTypes = map[string]bool{
	"DOMAIN": true, "DOMAIN-SUFFIX": true, "DOMAIN-KEYWORD": true, "DOMAIN-SET": true,
	"IP-CIDR": true, "IP-CIDR6": true, "IP-ASN": true, "GEOIP": true,
	"RULE-SET": true, "URL-REGEX": true, "USER-AGENT": true, "USER-AGENT-KEYWORD": true,
	"PROCESS-NAME": true, "SUBNET": true, "HOST": true, "HOST-SUFFIX": true,
	"HOST-KEYWORD": true, "HOST-WILDCARD": true, "DEST-PORT": true, "SRC-PORT": true,
	"IN-PORT": true, "PROTOCOL": true, "SCRIPT": true, "HTTP-RESPONSE": true,
	"AND": true, "OR": true, "NOT": true, "FINAL": true,
	"DIRECT": true, "REJECT": true, "REJECT-TINYGIF": true, "REJECT-DROP": true,
}

var confSections = map[string]bool{
	"GENERAL": true, "RULE": true, "PROXY": true, "POLICY": true,
	"HOST": true, "URL-REWRITE": true, "MITM": true, "SCRIPT": true,
	"FILTER": true, "REWRITE": true, "PANEL": true, "MANAGED-CONFIG": true,
}

// Validate checks Surge file content based on file extension.
func Validate(filename string, content []byte, strict bool) Result {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".list":
		return validateList(content, strict)
	case ".conf", ".module":
		return validateConf(content, strict)
	default:
		return Result{Valid: true, FileType: ext}
	}
}

// HasErrors returns true when result contains error-level issues.
func (r Result) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Level == LevelError {
			return true
		}
	}
	return false
}

func validateList(content []byte, strict bool) Result {
	res := Result{Valid: true, FileType: "list"}
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isComment(trimmed) {
			continue
		}
		issues := validateRuleLine(trimmed, i+1, strict)
		res.Issues = append(res.Issues, issues...)
	}
	res.Valid = !res.HasErrors()
	return res
}

func validateConf(content []byte, strict bool) Result {
	res := Result{Valid: true, FileType: "conf"}
	lines := strings.Split(string(content), "\n")
	section := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isComment(trimmed) {
			continue
		}

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = strings.ToUpper(strings.Trim(trimmed, "[]"))
			if strict && !confSections[section] {
				res.Issues = append(res.Issues, Issue{
					Line: i + 1, Level: LevelWarning,
					Message: fmt.Sprintf("unknown section [%s]", section),
				})
			}
			continue
		}

		if section == "RULE" || (section == "" && strings.Contains(trimmed, ",") && !strings.Contains(trimmed, "=")) {
			issues := validateRuleLine(trimmed, i+1, strict)
			res.Issues = append(res.Issues, issues...)
		} else if section == "" && strings.Contains(trimmed, "=") {
			res.Issues = append(res.Issues, Issue{
				Line: i + 1, Level: LevelWarning,
				Message: "key-value line outside any section",
			})
		} else if section != "" && section != "RULE" && !strings.Contains(trimmed, "=") {
			res.Issues = append(res.Issues, Issue{
				Line: i + 1, Level: LevelWarning,
				Message: "expected key = value format in [" + section + "]",
			})
		}
	}

	res.Valid = !res.HasErrors()
	return res
}

func validateRuleLine(line string, lineNo int, strict bool) []Issue {
	var issues []Issue

	if !strings.Contains(line, ",") {
		issues = append(issues, Issue{
			Line: lineNo, Level: LevelError,
			Message: "rule must be comma-separated (TYPE,param,policy)",
		})
		return issues
	}

	parts := strings.Split(line, ",")
	ruleType := strings.TrimSpace(strings.ToUpper(parts[0]))

	if ruleType == "" {
		issues = append(issues, Issue{
			Line: lineNo, Level: LevelError,
			Message: "empty rule type",
		})
		return issues
	}

	if !ruleTypes[ruleType] {
		level := LevelWarning
		if strict {
			level = LevelError
		}
		issues = append(issues, Issue{
			Line: lineNo, Level: level,
			Message: fmt.Sprintf("unknown rule type: %s", ruleType),
		})
	}

	switch ruleType {
	case "FINAL":
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			issues = append(issues, Issue{
				Line: lineNo, Level: LevelError,
				Message: "FINAL requires a policy (e.g. FINAL,DIRECT)",
			})
		}
	case "RULE-SET", "DOMAIN-SET":
		if len(parts) < 3 {
			issues = append(issues, Issue{
				Line: lineNo, Level: LevelError,
				Message: ruleType + " requires URL/path and policy",
			})
		} else {
			target := strings.TrimSpace(parts[1])
			if target == "" {
				issues = append(issues, Issue{
					Line: lineNo, Level: LevelError,
					Message: ruleType + " requires a non-empty URL or path",
				})
			}
		}
	case "IP-CIDR", "IP-CIDR6":
		if len(parts) < 3 {
			issues = append(issues, Issue{
				Line: lineNo, Level: LevelError,
				Message: ruleType + " requires CIDR and policy",
			})
		} else if !looksLikeCIDR(strings.TrimSpace(parts[1])) {
			issues = append(issues, Issue{
				Line: lineNo, Level: LevelWarning,
				Message: "IP-CIDR parameter does not look like a CIDR block",
			})
		}
	case "GEOIP":
		if len(parts) < 3 {
			issues = append(issues, Issue{
				Line: lineNo, Level: LevelError,
				Message: "GEOIP requires country code and policy",
			})
		}
	default:
		if len(parts) < 2 {
			issues = append(issues, Issue{
				Line: lineNo, Level: LevelError,
				Message: ruleType + " requires at least one parameter",
			})
		}
		if len(parts) >= 2 && strings.TrimSpace(parts[len(parts)-1]) == "" {
			issues = append(issues, Issue{
				Line: lineNo, Level: LevelError,
				Message: "missing policy at end of rule",
			})
		}
	}

	return issues
}

func isComment(line string) bool {
	return strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "//")
}

func looksLikeCIDR(s string) bool {
	if s == "" {
		return false
	}
	if !strings.Contains(s, "/") && !strings.Contains(s, ":") {
		// single IP without prefix length — Surge allows this
		return strings.Count(s, ".") >= 1 || strings.Contains(s, ":")
	}
	before, after, ok := strings.Cut(s, "/")
	if !ok {
		return false
	}
	if before == "" || after == "" {
		return false
	}
	for _, c := range after {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}