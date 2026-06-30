package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// Level 表示校验问题的严重级别。
const (
	LevelError   = "error"
	LevelWarning = "warning"
)

// Issue 描述单条校验问题。
type Issue struct {
	Line    int    `json:"line"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
	Level   string `json:"level"`
}

// Result 描述一次校验结果。
type Result struct {
	Valid    bool    `json:"valid"`
	FileType string  `json:"file_type"`
	Issues   []Issue `json:"issues"`
}

// ValidationError 表示保存时的阻断性校验失败。
type ValidationError struct {
	Result Result
}

func (e *ValidationError) Error() string {
	if len(e.Result.Issues) == 0 {
		return "校验失败"
	}
	return fmt.Sprintf("校验失败：%s", e.Result.Issues[0].Message)
}

var (
	yamlLinePattern = regexp.MustCompile(`line (\d+)`)

	ruleTypes = map[string]bool{
		"DOMAIN": true, "DOMAIN-SUFFIX": true, "DOMAIN-KEYWORD": true, "DOMAIN-SET": true,
		"IP-CIDR": true, "IP-CIDR6": true, "IP-ASN": true, "GEOIP": true,
		"RULE-SET": true, "URL-REGEX": true, "USER-AGENT": true, "USER-AGENT-KEYWORD": true,
		"PROCESS-NAME": true, "SUBNET": true, "HOST": true, "HOST-SUFFIX": true,
		"HOST-KEYWORD": true, "HOST-WILDCARD": true, "DEST-PORT": true, "SRC-PORT": true,
		"IN-PORT": true, "PROTOCOL": true, "SCRIPT": true, "HTTP-RESPONSE": true,
		"AND": true, "OR": true, "NOT": true, "FINAL": true,
		"DIRECT": true, "REJECT": true, "REJECT-TINYGIF": true, "REJECT-DROP": true,
	}

	confSections = map[string]bool{
		"GENERAL":        true,
		"RULE":           true,
		"PROXY":          true,
		"HOST":           true,
		"URL-REWRITE":    true,
		"HEADER-REWRITE": true,
		"MAP-LOCAL":      true,
		"SCRIPT":         true,
		"MITM":           true,
	}

	mihomoTopLevelKeys = map[string]bool{
		"port":            true,
		"socks-port":      true,
		"redir-port":      true,
		"mixed-port":      true,
		"tproxy-port":     true,
		"mode":            true,
		"allow-lan":       true,
		"log-level":       true,
		"dns":             true,
		"hosts":           true,
		"tun":             true,
		"proxies":         true,
		"proxy-groups":    true,
		"proxy-providers": true,
		"rule-providers":  true,
		"rules":           true,
		"sub-rules":       true,
	}

	singBoxStrongKeys = map[string]bool{
		"inbounds":     true,
		"outbounds":    true,
		"route":        true,
		"experimental": true,
		"endpoints":    true,
	}

	singBoxSoftKeys = map[string]bool{
		"log":      true,
		"dns":      true,
		"ntp":      true,
		"services": true,
	}
)

// HasErrors 返回结果中是否存在 error 级别问题。
func (r Result) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Level == LevelError {
			return true
		}
	}
	return false
}

// Validate 根据扩展名执行对应的配置校验。
func Validate(filename string, content []byte, strict bool) Result {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".list":
		return validateList(content, strict)
	case ".conf", ".module":
		return validateConf(content, strict)
	case ".yaml", ".yml":
		return validateYAMLConfig(content, strict)
	case ".json":
		return validateJSONConfig(content, strict)
	default:
		return Result{Valid: true, FileType: "plain"}
	}
}

func validateList(content []byte, strict bool) Result {
	res := Result{Valid: true, FileType: "surge-list"}
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
	res := Result{Valid: true, FileType: "surge-conf"}
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
					Line:    i + 1,
					Level:   LevelWarning,
					Message: fmt.Sprintf("未知的 Surge 配置段：[%s]", section),
				})
			}
			continue
		}

		if section == "RULE" || (section == "" && strings.Contains(trimmed, ",") && !strings.Contains(trimmed, "=")) {
			issues := validateRuleLine(trimmed, i+1, strict)
			res.Issues = append(res.Issues, issues...)
			continue
		}

		if section == "" && strings.Contains(trimmed, "=") {
			res.Issues = append(res.Issues, Issue{
				Line:    i + 1,
				Level:   LevelWarning,
				Message: "键值对位于任何配置段之外",
			})
			continue
		}

		if section != "" && section != "RULE" && !strings.Contains(trimmed, "=") {
			res.Issues = append(res.Issues, Issue{
				Line:    i + 1,
				Level:   LevelWarning,
				Message: fmt.Sprintf("[%s] 段内建议使用 key = value 格式", section),
			})
		}
	}

	res.Valid = !res.HasErrors()
	return res
}

func validateYAMLConfig(content []byte, strict bool) Result {
	res := Result{Valid: true, FileType: "yaml"}
	if len(bytes.TrimSpace(content)) == 0 {
		return res
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		res.Issues = append(res.Issues, yamlSyntaxIssue(err))
		res.Valid = false
		return res
	}

	doc := unwrapYAMLDocument(&root)
	if doc == nil {
		return res
	}
	if doc.Kind != yaml.MappingNode {
		res.Issues = append(res.Issues, Issue{
			Line:    doc.Line,
			Column:  doc.Column,
			Level:   LevelWarning,
			Message: "YAML 顶层建议使用对象映射结构",
		})
		res.Valid = !res.HasErrors()
		return res
	}

	fields := yamlFields(doc)
	switch {
	case isMihomoRuleset(fields):
		res.FileType = "meta-ruleset"
		validateMihomoRuleset(&res, fields)
	case looksLikeMihomoConfig(fields):
		res.FileType = "meta-yaml"
		validateMihomoConfig(&res, fields)
	default:
		if strict {
			res.Issues = append(res.Issues, Issue{
				Line:    doc.Line,
				Column:  doc.Column,
				Level:   LevelWarning,
				Message: "未识别为 Meta/Mihomo 配置，已仅执行 YAML 语法校验",
			})
		}
	}

	res.Valid = !res.HasErrors()
	return res
}

func validateJSONConfig(content []byte, strict bool) Result {
	res := Result{Valid: true, FileType: "json"}
	if len(bytes.TrimSpace(content)) == 0 {
		return res
	}

	var root any
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		res.Issues = append(res.Issues, jsonSyntaxIssue(content, err))
		res.Valid = false
		return res
	}

	object, ok := root.(map[string]any)
	if !ok {
		res.Issues = append(res.Issues, Issue{
			Level:   LevelWarning,
			Message: "JSON 顶层建议使用对象结构",
		})
		res.Valid = !res.HasErrors()
		return res
	}

	if looksLikeSingBoxConfig(object) {
		res.FileType = "sing-box-json"
		validateSingBoxConfig(&res, object, strict)
	} else if strict {
		res.Issues = append(res.Issues, Issue{
			Level:   LevelWarning,
			Message: "未识别为 sing-box 配置，已仅执行 JSON 语法校验",
		})
	}

	res.Valid = !res.HasErrors()
	return res
}

func validateMihomoRuleset(res *Result, fields map[string]*yaml.Node) {
	payload := fields["payload"]
	expectYAMLKind(res, payload, yaml.SequenceNode, "payload 必须是数组")
	if payload == nil || payload.Kind != yaml.SequenceNode {
		return
	}

	for _, item := range payload.Content {
		if item.Kind != yaml.ScalarNode {
			res.Issues = append(res.Issues, Issue{
				Line:    item.Line,
				Column:  item.Column,
				Level:   LevelWarning,
				Message: "payload 中的每一项建议为字符串规则",
			})
		}
	}
}

func validateMihomoConfig(res *Result, fields map[string]*yaml.Node) {
	expectYAMLKind(res, fields["proxies"], yaml.SequenceNode, "proxies 必须是数组")
	expectYAMLKind(res, fields["proxy-groups"], yaml.SequenceNode, "proxy-groups 必须是数组")
	expectYAMLKind(res, fields["rules"], yaml.SequenceNode, "rules 必须是数组")
	expectYAMLKind(res, fields["proxy-providers"], yaml.MappingNode, "proxy-providers 必须是对象")
	expectYAMLKind(res, fields["rule-providers"], yaml.MappingNode, "rule-providers 必须是对象")
	expectYAMLKind(res, fields["dns"], yaml.MappingNode, "dns 必须是对象")

	validateYAMLScalarSequence(res, fields["rules"], "rules")
}

func validateSingBoxConfig(res *Result, object map[string]any, strict bool) {
	expectJSONArray(res, object, "inbounds")
	expectJSONArray(res, object, "outbounds")
	expectJSONObject(res, object, "route")
	expectJSONObject(res, object, "dns")
	expectJSONObject(res, object, "log")
	expectJSONObject(res, object, "ntp")
	expectJSONObject(res, object, "experimental")

	if strict {
		if _, ok := object["outbounds"]; !ok {
			res.Issues = append(res.Issues, Issue{
				Level:   LevelWarning,
				Message: "sing-box 配置通常应包含 outbounds",
			})
		}
	}
}

func validateRuleLine(line string, lineNo int, strict bool) []Issue {
	var issues []Issue

	if !strings.Contains(line, ",") {
		return []Issue{{
			Line:    lineNo,
			Level:   LevelError,
			Message: "规则必须使用逗号分隔，例如 TYPE,param,policy",
		}}
	}

	parts := strings.Split(line, ",")
	ruleType := strings.TrimSpace(strings.ToUpper(parts[0]))
	if ruleType == "" {
		return []Issue{{
			Line:    lineNo,
			Level:   LevelError,
			Message: "规则类型不能为空",
		}}
	}

	if !ruleTypes[ruleType] {
		level := LevelWarning
		if strict {
			level = LevelError
		}
		issues = append(issues, Issue{
			Line:    lineNo,
			Level:   level,
			Message: fmt.Sprintf("未知的 Surge 规则类型：%s", ruleType),
		})
	}

	switch ruleType {
	case "FINAL":
		if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
			issues = append(issues, Issue{
				Line:    lineNo,
				Level:   LevelError,
				Message: "FINAL 必须带策略，例如 FINAL,DIRECT",
			})
		}
	case "RULE-SET", "DOMAIN-SET":
		if len(parts) < 3 {
			issues = append(issues, Issue{
				Line:    lineNo,
				Level:   LevelError,
				Message: ruleType + " 需要 URL/路径和策略",
			})
		} else if strings.TrimSpace(parts[1]) == "" {
			issues = append(issues, Issue{
				Line:    lineNo,
				Level:   LevelError,
				Message: ruleType + " 的 URL/路径不能为空",
			})
		}
	case "IP-CIDR", "IP-CIDR6":
		if len(parts) < 3 {
			issues = append(issues, Issue{
				Line:    lineNo,
				Level:   LevelError,
				Message: ruleType + " 需要 CIDR 和策略",
			})
		} else if !looksLikeCIDR(strings.TrimSpace(parts[1])) {
			issues = append(issues, Issue{
				Line:    lineNo,
				Level:   LevelWarning,
				Message: "CIDR 参数格式看起来不正确",
			})
		}
	case "GEOIP":
		if len(parts) < 3 {
			issues = append(issues, Issue{
				Line:    lineNo,
				Level:   LevelError,
				Message: "GEOIP 需要国家代码和策略",
			})
		}
	default:
		if len(parts) < 2 {
			issues = append(issues, Issue{
				Line:    lineNo,
				Level:   LevelError,
				Message: ruleType + " 至少需要一个参数",
			})
		}
		if len(parts) >= 2 && strings.TrimSpace(parts[len(parts)-1]) == "" {
			issues = append(issues, Issue{
				Line:    lineNo,
				Level:   LevelError,
				Message: "规则尾部缺少策略",
			})
		}
	}

	return issues
}

func yamlSyntaxIssue(err error) Issue {
	issue := Issue{
		Level:   LevelError,
		Message: "YAML 语法错误：" + err.Error(),
	}

	match := yamlLinePattern.FindStringSubmatch(err.Error())
	if len(match) == 2 {
		if line, convErr := strconv.Atoi(match[1]); convErr == nil {
			issue.Line = line
		}
	}

	return issue
}

func jsonSyntaxIssue(content []byte, err error) Issue {
	issue := Issue{
		Level:   LevelError,
		Message: "JSON 语法错误：" + err.Error(),
	}

	switch e := err.(type) {
	case *json.SyntaxError:
		issue.Line, issue.Column = lineColumnFromOffset(content, e.Offset)
	case *json.UnmarshalTypeError:
		issue.Line, issue.Column = lineColumnFromOffset(content, e.Offset)
		issue.Message = fmt.Sprintf("JSON 字段类型错误：期望 %s", e.Type.String())
	}

	return issue
}

func unwrapYAMLDocument(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func yamlFields(node *yaml.Node) map[string]*yaml.Node {
	fields := make(map[string]*yaml.Node, len(node.Content)/2)
	for i := 0; i+1 < len(node.Content); i += 2 {
		fields[strings.ToLower(strings.TrimSpace(node.Content[i].Value))] = node.Content[i+1]
	}
	return fields
}

func isMihomoRuleset(fields map[string]*yaml.Node) bool {
	_, ok := fields["payload"]
	return ok
}

func looksLikeMihomoConfig(fields map[string]*yaml.Node) bool {
	for key := range fields {
		if mihomoTopLevelKeys[key] {
			return true
		}
	}
	return false
}

func looksLikeSingBoxConfig(object map[string]any) bool {
	score := 0
	for key := range object {
		lower := strings.ToLower(key)
		if singBoxStrongKeys[lower] {
			return true
		}
		if singBoxSoftKeys[lower] {
			score++
		}
	}
	return score >= 2
}

func expectYAMLKind(res *Result, node *yaml.Node, kind yaml.Kind, message string) {
	if node == nil {
		return
	}
	if node.Kind == kind {
		return
	}
	res.Issues = append(res.Issues, Issue{
		Line:    node.Line,
		Column:  node.Column,
		Level:   LevelError,
		Message: message,
	})
}

func validateYAMLScalarSequence(res *Result, node *yaml.Node, field string) {
	if node == nil || node.Kind != yaml.SequenceNode {
		return
	}
	for _, item := range node.Content {
		if item.Kind != yaml.ScalarNode {
			res.Issues = append(res.Issues, Issue{
				Line:    item.Line,
				Column:  item.Column,
				Level:   LevelWarning,
				Message: fmt.Sprintf("%s 中的每一项建议为纯文本条目", field),
			})
		}
	}
}

func expectJSONArray(res *Result, object map[string]any, key string) {
	value, ok := object[key]
	if !ok {
		return
	}
	if _, ok := value.([]any); ok {
		return
	}
	res.Issues = append(res.Issues, Issue{
		Level:   LevelError,
		Message: key + " 必须是数组",
	})
}

func expectJSONObject(res *Result, object map[string]any, key string) {
	value, ok := object[key]
	if !ok {
		return
	}
	if _, ok := value.(map[string]any); ok {
		return
	}
	res.Issues = append(res.Issues, Issue{
		Level:   LevelError,
		Message: key + " 必须是对象",
	})
}

func lineColumnFromOffset(content []byte, offset int64) (int, int) {
	if offset <= 0 {
		return 0, 0
	}
	if offset > int64(len(content)) {
		offset = int64(len(content))
	}

	line := 1
	column := 1
	for i, b := range content[:offset-1] {
		if b == '\n' {
			line++
			column = 1
			continue
		}
		if i >= 0 {
			column++
		}
	}
	return line, column
}

func isComment(line string) bool {
	return strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "//")
}

func looksLikeCIDR(s string) bool {
	if s == "" {
		return false
	}
	if !strings.Contains(s, "/") && !strings.Contains(s, ":") {
		// Surge 允许不带掩码的单个 IP。
		return strings.Count(s, ".") >= 1 || strings.Contains(s, ":")
	}

	before, after, ok := strings.Cut(s, "/")
	if !ok || before == "" || after == "" {
		return false
	}
	for _, r := range after {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
