package engine

import (
	"fmt"

	"codeberg.org/xchangeee/desync/internal/api"
	"codeberg.org/xchangeee/desync/internal/config"
)

const defaultTTL = 3600

// FetchState reads the complete current state from the API and returns it as a
// Config ready to be written to disk (the init subcommand).
//
// Token policies: tokens with at least one scoping policy are included.
// Tokens without policies are skipped — an empty policy list in the state file
// means "delete all" on the next apply.
//
// RRsets: every domain is included with all its RRsets nested inside.
// TTL values equal to the default (3600) are zeroed so they are omitted from
// the JSON output, keeping the file concise.
func FetchState(client *api.Client) (*config.Config, error) {
	cfg := &config.Config{}

	if err := fetchTokenPolicies(client, cfg); err != nil {
		return nil, err
	}
	if err := fetchDomains(client, cfg); err != nil {
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
			continue
		}

		tp := config.TokenPolicies{TokenID: t.ID}
		for _, p := range policies {
			tp.Policies = append(tp.Policies, config.Policy{
				Domain:    p.Domain,
				Subname:   p.Subname,
				Type:      p.Type,
				PermWrite: p.PermWrite,
			})
		}
		cfg.TokenPolicies = append(cfg.TokenPolicies, tp)
	}

	return nil
}

func fetchDomains(client *api.Client, cfg *config.Config) error {
	domains, err := client.ListDomains()
	if err != nil {
		return fmt.Errorf("listing domains: %w", err)
	}

	for _, d := range domains {
		rrsets, err := client.ListRRsets(d.Name)
		if err != nil {
			return fmt.Errorf("listing rrsets for %s: %w", d.Name, err)
		}

		dom := config.Domain{Name: d.Name}
		for _, r := range rrsets {
			if managedByDesec[r.Type] {
				continue
			}
			ttl := r.TTL
			if ttl == defaultTTL {
				ttl = 0 // omitted from JSON; EffectiveTTL() restores the default
			}
			dom.RRsets = append(dom.RRsets, config.RRset{
				Subname: r.Subname,
				Type:    r.Type,
				TTL:     ttl,
				Records: r.Records,
			})
		}
		cfg.Domains = append(cfg.Domains, dom)
	}

	return nil
}
