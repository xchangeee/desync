package config_test

import (
	"testing"

	"github.com/spf13/afero"

	"codeberg.org/xchangeee/desync/internal/config"
)

// write puts content into an in-memory filesystem at path and returns the fs.
func write(t *testing.T, path, content string) afero.Fs {
	t.Helper()
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing test file: %v", err)
	}
	return fs
}

func TestLoad_HappyPath(t *testing.T) {
	domain := "example.com"
	fs := write(t, "state.json", `{
		"token_policies": [
			{
				"token_id": "aaaaaaaa-0000-0000-0000-000000000001",
				"policies": [
					{"domain": null, "subname": null, "type": null, "perm_write": false},
					{"domain": "example.com", "subname": null, "type": null, "perm_write": true}
				]
			}
		],
		"domains": [
			{
				"name": "example.com",
				"rrsets": [
					{"subname": "www", "type": "A", "records": ["1.2.3.4"]},
					{"subname": "www", "type": "AAAA", "ttl": 300, "records": ["::1"]}
				]
			}
		]
	}`)

	cfg, err := config.Load(fs, "state.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.TokenPolicies) != 1 {
		t.Fatalf("want 1 token policy group, got %d", len(cfg.TokenPolicies))
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0].Name != domain {
		t.Fatalf("want domain %q, got %+v", domain, cfg.Domains)
	}
	if len(cfg.Domains[0].RRsets) != 2 {
		t.Fatalf("want 2 rrsets, got %d", len(cfg.Domains[0].RRsets))
	}
}

func TestLoad_TTLDefault(t *testing.T) {
	fs := write(t, "s.json", `{
		"domains": [{"name": "x.de", "rrsets": [
			{"subname": "", "type": "A", "records": ["1.1.1.1"]}
		]}]
	}`)
	cfg, err := config.Load(fs, "s.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := cfg.Domains[0].RRsets[0]
	if r.TTL != 0 {
		t.Errorf("want raw TTL 0 (omitted), got %d", r.TTL)
	}
	if r.EffectiveTTL() != 3600 {
		t.Errorf("want effective TTL 3600, got %d", r.EffectiveTTL())
	}
}

func TestLoad_ExplicitTTL(t *testing.T) {
	fs := write(t, "s.json", `{
		"domains": [{"name": "x.de", "rrsets": [
			{"subname": "", "type": "A", "ttl": 60, "records": ["1.1.1.1"]}
		]}]
	}`)
	cfg, err := config.Load(fs, "s.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cfg.Domains[0].RRsets[0].EffectiveTTL(); got != 60 {
		t.Errorf("want 60, got %d", got)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	_, err := config.Load(fs, "missing.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_UnknownField(t *testing.T) {
	fs := write(t, "s.json", `{"unknown_key": true}`)
	_, err := config.Load(fs, "s.json")
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestLoad_DuplicateDomain(t *testing.T) {
	fs := write(t, "s.json", `{
		"domains": [
			{"name": "dup.de", "rrsets": []},
			{"name": "dup.de", "rrsets": []}
		]
	}`)
	_, err := config.Load(fs, "s.json")
	if err == nil {
		t.Fatal("expected error for duplicate domain")
	}
}

func TestLoad_DuplicateRRset(t *testing.T) {
	fs := write(t, "s.json", `{
		"domains": [{"name": "x.de", "rrsets": [
			{"subname": "www", "type": "A", "records": ["1.1.1.1"]},
			{"subname": "www", "type": "A", "records": ["2.2.2.2"]}
		]}]
	}`)
	_, err := config.Load(fs, "s.json")
	if err == nil {
		t.Fatal("expected error for duplicate rrset")
	}
}

func TestLoad_DuplicateTokenID(t *testing.T) {
	fs := write(t, "s.json", `{
		"token_policies": [
			{"token_id": "aaaa", "policies": []},
			{"token_id": "aaaa", "policies": []}
		]
	}`)
	_, err := config.Load(fs, "s.json")
	if err == nil {
		t.Fatal("expected error for duplicate token_id")
	}
}

func TestLoad_RejectsNSType(t *testing.T) {
	fs := write(t, "s.json", `{
		"domains": [{"name": "x.de", "rrsets": [
			{"subname": "", "type": "NS", "records": ["ns1.desec.io."]}
		]}]
	}`)
	_, err := config.Load(fs, "s.json")
	if err == nil {
		t.Fatal("expected error for NS type")
	}
}

func TestLoad_RejectsSOAType(t *testing.T) {
	fs := write(t, "s.json", `{
		"domains": [{"name": "x.de", "rrsets": [
			{"subname": "", "type": "SOA", "records": ["ns1. admin. 1 2 3 4 5"]}
		]}]
	}`)
	_, err := config.Load(fs, "s.json")
	if err == nil {
		t.Fatal("expected error for SOA type")
	}
}

func TestLoad_RejectsZeroTTL(t *testing.T) {
	fs := write(t, "s.json", `{
		"domains": [{"name": "x.de", "rrsets": [
			{"subname": "", "type": "A", "ttl": -1, "records": ["1.1.1.1"]}
		]}]
	}`)
	_, err := config.Load(fs, "s.json")
	if err == nil {
		t.Fatal("expected error for negative TTL")
	}
}
