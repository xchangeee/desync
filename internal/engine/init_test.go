package engine_test

import (
	"testing"

	"github.com/xchangeee/desync/internal/api"
	"github.com/xchangeee/desync/internal/engine"
)

func TestFetchState_TokensWithoutPoliciesSkipped(t *testing.T) {
	stub := &stubClient{
		listTokensFn: func() ([]api.Token, error) {
			return []api.Token{
				{ID: "no-policies", Name: "bare"},
				{ID: "has-policies", Name: "scoped"},
			}, nil
		},
		listPoliciesFn: func(tokenID string) ([]api.TokenPolicy, error) {
			if tokenID == "has-policies" {
				return []api.TokenPolicy{
					{ID: "p1", Domain: nil, Subname: nil, Type: nil, PermWrite: false},
				}, nil
			}
			return nil, nil // no-policies token has none
		},
		listDomainsFn: func() ([]api.Domain, error) { return nil, nil },
	}

	cfg, err := engine.FetchState(stub)
	if err != nil {
		t.Fatalf("FetchState: %v", err)
	}
	if len(cfg.TokenPolicies) != 1 {
		t.Fatalf("want 1 token policy group (skipping token without policies), got %d", len(cfg.TokenPolicies))
	}
	if cfg.TokenPolicies[0].TokenID != "has-policies" {
		t.Errorf("want token has-policies, got %s", cfg.TokenPolicies[0].TokenID)
	}
}

func TestFetchState_NSAndSOAFiltered(t *testing.T) {
	stub := &stubClient{
		listTokensFn: func() ([]api.Token, error) { return nil, nil },
		listDomainsFn: func() ([]api.Domain, error) {
			return []api.Domain{{Name: "example.com"}}, nil
		},
		listRRsetsFn: func(string) ([]api.RRset, error) {
			return []api.RRset{
				{Subname: "", Type: "NS", TTL: 3600, Records: []string{"ns1.desec.io."}},
				{Subname: "", Type: "SOA", TTL: 3600, Records: []string{"ns1. admin. 1 2 3 4 5"}},
				{Subname: "", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
			}, nil
		},
	}

	cfg, err := engine.FetchState(stub)
	if err != nil {
		t.Fatalf("FetchState: %v", err)
	}
	if len(cfg.Domains) != 1 {
		t.Fatalf("want 1 domain, got %d", len(cfg.Domains))
	}
	rrsets := cfg.Domains[0].RRsets
	if len(rrsets) != 1 {
		t.Fatalf("want 1 rrset (NS and SOA filtered), got %d: %+v", len(rrsets), rrsets)
	}
	if rrsets[0].Type != "A" {
		t.Errorf("want A record, got %s", rrsets[0].Type)
	}
}

func TestFetchState_TTL3600WrittenAsZero(t *testing.T) {
	stub := &stubClient{
		listTokensFn: func() ([]api.Token, error) { return nil, nil },
		listDomainsFn: func() ([]api.Domain, error) {
			return []api.Domain{{Name: "example.com"}}, nil
		},
		listRRsetsFn: func(string) ([]api.RRset, error) {
			return []api.RRset{
				{Subname: "www", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
			}, nil
		},
	}

	cfg, err := engine.FetchState(stub)
	if err != nil {
		t.Fatalf("FetchState: %v", err)
	}
	r := cfg.Domains[0].RRsets[0]
	if r.TTL != 0 {
		t.Errorf("TTL 3600 should be stored as 0 (omitempty), got %d", r.TTL)
	}
	if r.EffectiveTTL() != 3600 {
		t.Errorf("EffectiveTTL should restore to 3600, got %d", r.EffectiveTTL())
	}
}

func TestFetchState_NonDefaultTTLPreserved(t *testing.T) {
	stub := &stubClient{
		listTokensFn: func() ([]api.Token, error) { return nil, nil },
		listDomainsFn: func() ([]api.Domain, error) {
			return []api.Domain{{Name: "example.com"}}, nil
		},
		listRRsetsFn: func(string) ([]api.RRset, error) {
			return []api.RRset{
				{Subname: "_acme", Type: "TXT", TTL: 60, Records: []string{"\"challenge\""}},
			}, nil
		},
	}

	cfg, err := engine.FetchState(stub)
	if err != nil {
		t.Fatalf("FetchState: %v", err)
	}
	if got := cfg.Domains[0].RRsets[0].TTL; got != 60 {
		t.Errorf("want TTL 60 preserved, got %d", got)
	}
}

func TestFetchState_PoliciesMappedCorrectly(t *testing.T) {
	dom := "example.com"
	stub := &stubClient{
		listTokensFn: func() ([]api.Token, error) {
			return []api.Token{{ID: "t1", Name: "mytoken"}}, nil
		},
		listPoliciesFn: func(string) ([]api.TokenPolicy, error) {
			return []api.TokenPolicy{
				{ID: "p1", Domain: nil, Subname: nil, Type: nil, PermWrite: false},
				{ID: "p2", Domain: &dom, Subname: nil, Type: nil, PermWrite: true},
			}, nil
		},
		listDomainsFn: func() ([]api.Domain, error) { return nil, nil },
	}

	cfg, err := engine.FetchState(stub)
	if err != nil {
		t.Fatalf("FetchState: %v", err)
	}
	if len(cfg.TokenPolicies) != 1 {
		t.Fatalf("want 1 token policy group, got %d", len(cfg.TokenPolicies))
	}
	tp := cfg.TokenPolicies[0]
	if tp.TokenID != "t1" {
		t.Errorf("want tokenId t1, got %s", tp.TokenID)
	}
	if len(tp.Policies) != 2 {
		t.Fatalf("want 2 policies, got %d", len(tp.Policies))
	}
	if tp.Policies[1].Domain == nil || *tp.Policies[1].Domain != dom {
		t.Errorf("want second policy domain %s, got %v", dom, tp.Policies[1].Domain)
	}
	if !tp.Policies[1].PermWrite {
		t.Error("want permWrite true on specific policy")
	}
}
