package engine

import (
	"fmt"

	"codeberg.org/xchangeee/desync/internal/api"
	"codeberg.org/xchangeee/desync/internal/config"
)

// FetchState reads the complete current state from the API and returns it as a
// Config that can be written to disk. It is the backing implementation of the
// init subcommand.
//
// Token policies: every token that has at least one scoping policy is included.
// Tokens without policies are skipped — an empty policy list in the state file
// would be interpreted as "delete all policies" on the next apply.
//
// RRsets: every domain in the account is included with all its RRsets.
func FetchState(client *api.Client) (*config.Config, error) {
	cfg := &config.Config{}

	if err := fetchTokenPolicies(client, cfg); err != nil {
		return nil, err
	}
	if err := fetchRRsets(client, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func fetchTokenPolicies(client *api.Client, cfg *config.Config) error {
	tokens, err := client.ListTokens()
	if err != nil {
		return fmt.Errorf("listing tokens: %w", err)
	}

	for _, t := range tokens {
		policies, err := client.ListPolicies(t.ID)
		if err != nil {
			return fmt.Errorf("listing policies for token %s (%s): %w", t.ID, t.Name, err)
		}
		if len(policies) == 0 {
			// Skip: an empty list in the state file means "delete all", which
			// would be a no-op now but dangerous after any manual change.
			continue
		}

		tp := config.TokenPolicies{TokenID: t.ID}
		for _, p := range policies {
			pol := config.Policy{
				Domain:    p.Domain,
				Subname:   p.Subname,
				Type:      p.Type,
				PermWrite: p.PermWrite,
			}
			tp.Policies = append(tp.Policies, pol)
		}
		cfg.TokenPolicies = append(cfg.TokenPolicies, tp)
	}

	return nil
}

func fetchRRsets(client *api.Client, cfg *config.Config) error {
	domains, err := client.ListDomains()
	if err != nil {
		return fmt.Errorf("listing domains: %w", err)
	}

	for _, d := range domains {
		rrsets, err := client.ListRRsets(d.Name)
		if err != nil {
			return fmt.Errorf("listing rrsets for %s: %w", d.Name, err)
		}
		for _, r := range rrsets {
			cfg.RRsets = append(cfg.RRsets, config.RRset{
				Domain:  d.Name,
				Subname: r.Subname,
				Type:    r.Type,
				TTL:     r.TTL,
				Records: r.Records,
			})
		}
	}

	return nil
}
