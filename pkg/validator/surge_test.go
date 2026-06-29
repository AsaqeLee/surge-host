package validator

import "testing"

func TestValidateList_Valid(t *testing.T) {
	content := []byte(`# comment
DOMAIN-SUFFIX,google.com,PROXY
GEOIP,CN,DIRECT
FINAL,DIRECT
`)
	r := Validate("rules.list", content, false)
	if !r.Valid {
		t.Fatalf("expected valid, got issues: %+v", r.Issues)
	}
}

func TestValidateList_InvalidRule(t *testing.T) {
	content := []byte("DOMAIN-SUFFIX\n")
	r := Validate("rules.list", content, false)
	if r.Valid {
		t.Fatal("expected invalid")
	}
	if len(r.Issues) == 0 || r.Issues[0].Level != LevelError {
		t.Fatalf("expected error issue, got %+v", r.Issues)
	}
}

func TestValidateList_UnknownTypeStrict(t *testing.T) {
	content := []byte("UNKNOWN-TYPE,test.com,PROXY\n")
	r := Validate("rules.list", content, true)
	if r.Valid {
		t.Fatal("expected invalid in strict mode")
	}
}

func TestValidateConf_Valid(t *testing.T) {
	content := []byte(`[General]
loglevel = notify

[Rule]
DOMAIN-SUFFIX,apple.com,DIRECT
FINAL,PROXY
`)
	r := Validate("surge.conf", content, false)
	if !r.Valid {
		t.Fatalf("expected valid, got %+v", r.Issues)
	}
}

func TestValidateConf_RuleSetMissingPolicy(t *testing.T) {
	content := []byte(`[Rule]
RULE-SET,https://example.com/rules.list
`)
	r := Validate("surge.conf", content, false)
	if r.Valid {
		t.Fatal("expected invalid")
	}
}

func TestValidate_SkipsNonSurgeExt(t *testing.T) {
	r := Validate("readme.txt", []byte("anything"), true)
	if !r.Valid {
		t.Fatal("txt should skip validation")
	}
}