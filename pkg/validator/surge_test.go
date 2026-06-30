package validator

import "testing"

func TestValidateListValid(t *testing.T) {
	content := []byte("# comment\nDOMAIN-SUFFIX,google.com,PROXY\nGEOIP,CN,DIRECT\nFINAL,DIRECT\n")
	result := Validate("rules.list", content, false)
	if !result.Valid {
		t.Fatalf("expected valid result, got issues: %+v", result.Issues)
	}
}

func TestValidateListInvalidRule(t *testing.T) {
	content := []byte("DOMAIN-SUFFIX\n")
	result := Validate("rules.list", content, false)
	if result.Valid {
		t.Fatal("expected invalid result")
	}
	if len(result.Issues) == 0 || result.Issues[0].Level != LevelError {
		t.Fatalf("expected error issue, got: %+v", result.Issues)
	}
}

func TestValidateConfRuleSetMissingPolicy(t *testing.T) {
	content := []byte("[Rule]\nRULE-SET,https://example.com/rules.list\n")
	result := Validate("surge.conf", content, false)
	if result.Valid {
		t.Fatal("expected invalid Surge config")
	}
}

func TestValidateMetaYAMLConfigValid(t *testing.T) {
	content := []byte(`
mixed-port: 7890
proxies:
  - name: test
    type: direct
proxy-groups:
  - name: auto
    type: select
    proxies:
      - test
rules:
  - MATCH,auto
`)

	result := Validate("meta.yaml", content, true)
	if !result.Valid {
		t.Fatalf("expected valid Meta YAML, got issues: %+v", result.Issues)
	}
	if result.FileType != "meta-yaml" {
		t.Fatalf("expected file type meta-yaml, got %s", result.FileType)
	}
}

func TestValidateMetaRulesetInvalidPayload(t *testing.T) {
	content := []byte("payload: invalid\n")
	result := Validate("provider.yaml", content, true)
	if result.Valid {
		t.Fatal("expected invalid Meta ruleset")
	}
}

func TestValidateSingBoxJSONValid(t *testing.T) {
	content := []byte(`{
  "log": { "level": "info" },
  "inbounds": [],
  "outbounds": [
    { "type": "direct", "tag": "direct" }
  ],
  "route": {}
}`)

	result := Validate("sing-box.json", content, true)
	if !result.Valid {
		t.Fatalf("expected valid sing-box JSON, got issues: %+v", result.Issues)
	}
	if result.FileType != "sing-box-json" {
		t.Fatalf("expected file type sing-box-json, got %s", result.FileType)
	}
}

func TestValidateSingBoxJSONInvalidSyntax(t *testing.T) {
	content := []byte(`{"outbounds":[}`)
	result := Validate("sing-box.json", content, true)
	if result.Valid {
		t.Fatal("expected invalid sing-box JSON")
	}
	if len(result.Issues) == 0 || result.Issues[0].Level != LevelError {
		t.Fatalf("expected error issue, got: %+v", result.Issues)
	}
}

func TestValidateSkipsPlainText(t *testing.T) {
	result := Validate("readme.txt", []byte("anything"), true)
	if !result.Valid {
		t.Fatal("txt should skip validation")
	}
}
