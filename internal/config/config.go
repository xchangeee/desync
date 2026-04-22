// Package config defines the declarative desired-state file format and loads it.
//
// desync manages two resource types:
//
//   - token_policies: scoping policies on existing deSEC API tokens, referenced
//     by token UUID. desync fully reconciles the policy set for every listed
//     token: policies in the file are created or updated, and policies that
//     exist in the API but are absent from the file are deleted.
//
//   - rrsets: DNS resource record sets, grouped by domain. desync fully
//     reconciles every domain that appears in this section: RRsets in the file
//     are created or updated, and RRsets that exist in the API but are absent
//     from the file are deleted. Domains themselves are never created or
//     deleted by desync — they must already exist.
//
// Tokens and domains are intentionally not managed here:
//   - Token secrets are shown only once at creation time, making declarative
//     token creation impractical.
//   - Domains have no mutable fields and should not be auto-deleted;
//     creation is done manually or via the deSEC web UI.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is the top-level structure of a desync state file.
type Config struct {
	TokenPolicies []TokenPolicies `json:"token_policies"`
	RRsets        []RRset         `json:"rrsets"`
}

// TokenPolicies declares the complete set of desired scoping policies for a
// single token, identified by its UUID (the "id" field in the deSEC API, not
// the secret token value).
type TokenPolicies struct {
	// TokenID is the UUID of the existing token to configure.
	TokenID string `json:"token_id"`
	// Policies is the exhaustive desired policy list for this token.
	// The engine will create missing policies, update changed ones, and delete
	// any policy that is present in the API but absent from this list.
	Policies []Policy `json:"policies"`
}

// Policy describes a desired token scoping policy.
// The (Domain, Subname, Type) triple is the identity key; nil means wildcard.
//
// The deSEC API enforces that the default policy (all three fields null) must
// be created before any specific policy, and deleted last. desync handles
// this ordering automatically during apply.
type Policy struct {
	Domain    *string `json:"domain"`
	Subname   *string `json:"subname"`
	Type      *string `json:"type"`
	PermWrite bool    `json:"perm_write"`
}

// RRset describes a desired DNS resource record set.
// Domain must already exist in the account. Subname is the DNS label relative
// to the domain apex; an empty string refers to the apex itself.
type RRset struct {
	Domain  string   `json:"domain"`
	Subname string   `json:"subname"`
	Type    string   `json:"type"`
	TTL     int      `json:"ttl"`
	Records []string `json:"records"`
}

// Load reads and parses the JSON state file at path, returning a validated Config.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	dec := json.NewDecoder(f)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &cfg, nil
}

// validate performs basic structural checks on the config.
func validate(cfg *Config) error {
	seenToken := make(map[string]bool)
	for _, tp := range cfg.TokenPolicies {
		if tp.TokenID == "" {
			return fmt.Errorf("token_policies entry must have a non-empty token_id")
		}
		if seenToken[tp.TokenID] {
			return fmt.Errorf("duplicate token_id %q in token_policies", tp.TokenID)
		}
		seenToken[tp.TokenID] = true
	}

	type rrKey struct{ domain, subname, rrtype string }
	seenR := make(map[rrKey]bool)
	for _, r := range cfg.RRsets {
		if r.Domain == "" {
			return fmt.Errorf("rrset domain must not be empty")
		}
		if r.Type == "" {
			return fmt.Errorf("rrset type must not be empty (domain %s subname %q)", r.Domain, r.Subname)
		}
		if r.TTL <= 0 {
			return fmt.Errorf("rrset ttl must be positive (domain %s %q %s)", r.Domain, r.Subname, r.Type)
		}
		k := rrKey{r.Domain, r.Subname, r.Type}
		if seenR[k] {
			return fmt.Errorf("duplicate rrset (%s %q %s)", r.Domain, r.Subname, r.Type)
		}
		seenR[k] = true
	}

	return nil
}
