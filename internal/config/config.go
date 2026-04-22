// Package config defines the declarative desired-state file format and loads it.
//
// desync manages two resource types:
//
//   - token_policies: scoping policies on existing deSEC API tokens, referenced
//     by token UUID. desync fully reconciles the policy set for every listed
//     token: policies in the file are created or updated, and policies that
//     exist in the API but are absent from the file are deleted.
//
//   - domains: DNS zones, each with a nested list of desired RRsets. desync
//     fully reconciles every domain's RRsets: entries in the file are created
//     or updated, and RRsets that exist in the API but are absent from the file
//     are deleted. Domains themselves are never created or deleted by desync —
//     they must already exist in the account.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is the top-level structure of a desync state file.
type Config struct {
	TokenPolicies []TokenPolicies `json:"token_policies"`
	Domains       []Domain        `json:"domains"`
}

// TokenPolicies declares the complete set of desired scoping policies for a
// single token, identified by its UUID.
type TokenPolicies struct {
	TokenID  string   `json:"token_id"`
	Policies []Policy `json:"policies"`
}

// Policy describes a desired token scoping policy.
// The (Domain, Subname, Type) triple is the identity key; nil means wildcard.
type Policy struct {
	Domain    *string `json:"domain"`
	Subname   *string `json:"subname"`
	Type      *string `json:"type"`
	PermWrite bool    `json:"perm_write"`
}

// Domain groups the desired RRsets for one DNS zone.
type Domain struct {
	Name   string  `json:"name"`
	RRsets []RRset `json:"rrsets"`
}

// RRset describes a desired DNS resource record set within a domain.
// TTL defaults to 3600 when omitted from the file.
type RRset struct {
	Subname string   `json:"subname"`
	Type    string   `json:"type"`
	TTL     int      `json:"ttl,omitempty"` // 0 means "use default (3600)"
	Records []string `json:"records"`
}

// EffectiveTTL returns the RRset's TTL, substituting the default of 3600
// when the field was omitted (parsed as 0).
func (r RRset) EffectiveTTL() int {
	if r.TTL == 0 {
		return 3600
	}
	return r.TTL
}

// Load reads and parses the JSON state file at path.
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

	seenDomain := make(map[string]bool)
	for _, d := range cfg.Domains {
		if d.Name == "" {
			return fmt.Errorf("domain name must not be empty")
		}
		if seenDomain[d.Name] {
			return fmt.Errorf("duplicate domain %q", d.Name)
		}
		seenDomain[d.Name] = true

		type rrKey struct{ subname, rrtype string }
		seenR := make(map[rrKey]bool)
		for _, r := range d.RRsets {
			if r.Type == "" {
				return fmt.Errorf("rrset type must not be empty (domain %s subname %q)", d.Name, r.Subname)
			}
			if r.Type == "NS" || r.Type == "SOA" {
				return fmt.Errorf("rrset type %s is managed by deSEC and cannot be declared in the state file (domain %s subname %q)", r.Type, d.Name, r.Subname)
			}
			if r.EffectiveTTL() <= 0 {
				return fmt.Errorf("rrset ttl must be positive (domain %s %q %s)", d.Name, r.Subname, r.Type)
			}
			k := rrKey{r.Subname, r.Type}
			if seenR[k] {
				return fmt.Errorf("duplicate rrset (%s %q %s)", d.Name, r.Subname, r.Type)
			}
			seenR[k] = true
		}
	}

	return nil
}
