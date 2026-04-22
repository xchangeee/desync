package engine_test

import (
	"testing"

	"codeberg.org/xchangeee/desync/internal/api"
	"codeberg.org/xchangeee/desync/internal/config"
	"codeberg.org/xchangeee/desync/internal/engine"
)

// TestApply_PolicyOrdering verifies that when a token has no current policies
// and the desired list contains a mix of specific and default policies, the
// default is always created first.
func TestApply_PolicyOrdering(t *testing.T) {
	dom := "example.com"
	var created []api.TokenPolicyWriteFields

	stub := &stubClient{
		listPoliciesFn: func(string) ([]api.TokenPolicy, error) { return nil, nil },
		createPolicyFn: func(_ string, f api.TokenPolicyWriteFields) (*api.TokenPolicy, error) {
			created = append(created, f)
			return &api.TokenPolicy{}, nil
		},
	}

	// Desired: specific policy listed BEFORE the default — engine must reorder.
	cfg := &config.Config{
		TokenPolicies: []config.TokenPolicies{
			{
				TokenID: "tok1",
				Policies: []config.Policy{
					{Domain: &dom, Subname: nil, Type: nil, PermWrite: true},
					{Domain: nil, Subname: nil, Type: nil, PermWrite: false},
				},
			},
		},
	}

	diff, err := engine.Diff(cfg, stub)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if err := engine.Apply(diff, cfg, stub); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if len(created) != 2 {
		t.Fatalf("want 2 policy creates, got %d", len(created))
	}
	// First created policy must be the default (all-null fields).
	if created[0].Domain != nil || created[0].Subname != nil || created[0].Type != nil {
		t.Errorf("first create should be default policy, got domain=%v subname=%v type=%v",
			created[0].Domain, created[0].Subname, created[0].Type)
	}
	// Second must be the specific policy.
	if created[1].Domain == nil || *created[1].Domain != dom {
		t.Errorf("second create should be specific policy for %s, got %v", dom, created[1].Domain)
	}
}

// TestApply_PolicyDeleteOrdering verifies that specific policies are deleted
// before the default policy.
func TestApply_PolicyDeleteOrdering(t *testing.T) {
	dom := "example.com"
	var deleted []string

	current := []api.TokenPolicy{
		{ID: "p-default", Domain: nil, Subname: nil, Type: nil, PermWrite: false},
		{ID: "p-specific", Domain: &dom, Subname: nil, Type: nil, PermWrite: true},
	}
	stub := &stubClient{
		listPoliciesFn: func(string) ([]api.TokenPolicy, error) { return current, nil },
		deletePolicyFn: func(_, policyID string) error {
			deleted = append(deleted, policyID)
			return nil
		},
	}

	// Desired: empty (delete everything).
	cfg := &config.Config{
		TokenPolicies: []config.TokenPolicies{
			{TokenID: "tok1", Policies: nil},
		},
	}

	diff, err := engine.Diff(cfg, stub)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if err := engine.Apply(diff, cfg, stub); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if len(deleted) != 2 {
		t.Fatalf("want 2 deletions, got %d", len(deleted))
	}
	if deleted[0] != "p-specific" {
		t.Errorf("specific policy should be deleted first, got %s", deleted[0])
	}
	if deleted[1] != "p-default" {
		t.Errorf("default policy should be deleted last, got %s", deleted[1])
	}
}

// TestApply_RRsetBulkPatch verifies that all RRset changes for a domain are
// sent as a single bulk PATCH with correct payloads.
func TestApply_RRsetBulkPatch(t *testing.T) {
	var patchedDomain string
	var patchedItems []api.RRsetWriteFields

	current := []api.RRset{
		{Subname: "www", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},  // update records
		{Subname: "old", Type: "CNAME", TTL: 3600, Records: []string{"x."}},  // delete
	}
	stub := &stubClient{
		listRRsetsFn: func(string) ([]api.RRset, error) { return current, nil },
		bulkPatchRRsetsFn: func(domain string, items []api.RRsetWriteFields) error {
			patchedDomain = domain
			patchedItems = items
			return nil
		},
	}

	cfg := &config.Config{
		Domains: []config.Domain{
			{
				Name: "example.com",
				RRsets: []config.RRset{
					{Subname: "www", Type: "A", Records: []string{"5.6.7.8"}}, // changed
					{Subname: "new", Type: "AAAA", Records: []string{"::1"}},  // create
					// "old" CNAME absent → delete
				},
			},
		},
	}

	diff, err := engine.Diff(cfg, stub)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if err := engine.Apply(diff, cfg, stub); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if patchedDomain != "example.com" {
		t.Errorf("want patch for example.com, got %q", patchedDomain)
	}
	if len(patchedItems) != 3 {
		t.Fatalf("want 3 patch items (update+create+delete), got %d", len(patchedItems))
	}

	byKey := make(map[string]api.RRsetWriteFields)
	for _, item := range patchedItems {
		byKey[item.Subname+"/"+item.Type] = item
	}

	if r, ok := byKey["www/A"]; !ok || r.Records[0] != "5.6.7.8" {
		t.Errorf("www/A should be updated to 5.6.7.8, got %+v", byKey["www/A"])
	}
	if r, ok := byKey["new/AAAA"]; !ok || r.Records[0] != "::1" {
		t.Errorf("new/AAAA should be created, got %+v", byKey["new/AAAA"])
	}
	if r, ok := byKey["old/CNAME"]; !ok || len(r.Records) != 0 {
		t.Errorf("old/CNAME should be deleted (empty records), got %+v", byKey["old/CNAME"])
	}
}

// TestApply_NoAPICallsWhenNoDiff verifies that a no-op plan makes no API calls.
func TestApply_NoAPICallsWhenNoDiff(t *testing.T) {
	calls := 0
	bump := func() { calls++ }

	current := []api.RRset{
		{Subname: "www", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
	}
	stub := &stubClient{
		listRRsetsFn:      func(string) ([]api.RRset, error) { return current, nil },
		bulkPatchRRsetsFn: func(string, []api.RRsetWriteFields) error { bump(); return nil },
	}

	cfg := &config.Config{
		Domains: []config.Domain{
			{Name: "example.com", RRsets: []config.RRset{
				{Subname: "www", Type: "A", Records: []string{"1.2.3.4"}},
			}},
		},
	}

	diff, err := engine.Diff(cfg, stub)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if err := engine.Apply(diff, cfg, stub); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if calls != 0 {
		t.Errorf("want 0 API calls for no-op diff, got %d", calls)
	}
}
