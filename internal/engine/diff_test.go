package engine_test

import (
	"testing"

	"github.com/xchangeee/desync/internal/api"
	"github.com/xchangeee/desync/internal/config"
	"github.com/xchangeee/desync/internal/engine"
)

// ---- withImplicitDefault ---------------------------------------------------

func TestWithImplicitDefault_NoOp_WhenEmpty(t *testing.T) {
	// Empty policy list → unchanged (no implicit default injected).
	cfg := config.TokenPolicies{TokenID: "t1", Policies: nil}
	result := diffPoliciesStub(t, cfg, nil)
	if len(result) != 0 {
		t.Fatalf("want 0 diffs for empty policy list, got %d", len(result))
	}
}

func TestWithImplicitDefault_Injected_WhenMissing(t *testing.T) {
	// Specific policy present, no default → implicit default is injected and
	// shows up as a ChangeCreate.
	dom := "example.com"
	cfg := config.TokenPolicies{
		TokenID: "t1",
		Policies: []config.Policy{
			{Domain: &dom, Subname: nil, Type: nil, PermWrite: true},
		},
	}
	result := diffPoliciesStub(t, cfg, nil) // no current policies
	if len(result) != 2 {
		t.Fatalf("want 2 diffs (implicit default + specific), got %d", len(result))
	}
	// First diff must be the implicit default create.
	if result[0].Kind != engine.ChangeCreate || result[0].Domain != nil {
		t.Errorf("first diff should be default policy create, got %+v", result[0])
	}
}

func TestWithImplicitDefault_NoOp_WhenExplicit(t *testing.T) {
	// Explicit default already present → not duplicated.
	dom := "example.com"
	cfg := config.TokenPolicies{
		TokenID: "t1",
		Policies: []config.Policy{
			{Domain: nil, Subname: nil, Type: nil, PermWrite: false},
			{Domain: &dom, Subname: nil, Type: nil, PermWrite: true},
		},
	}
	result := diffPoliciesStub(t, cfg, nil)
	defaults := 0
	for _, d := range result {
		if d.Domain == nil && d.Subname == nil && d.Type == nil {
			defaults++
		}
	}
	if defaults != 1 {
		t.Errorf("want exactly 1 default policy diff, got %d", defaults)
	}
}

// ---- policy diffs ----------------------------------------------------------

func TestDiff_PolicyCreate(t *testing.T) {
	dom := "example.com"
	cfg := config.TokenPolicies{
		TokenID: "t1",
		Policies: []config.Policy{
			{Domain: nil, Subname: nil, Type: nil, PermWrite: false},
			{Domain: &dom, Subname: nil, Type: nil, PermWrite: true},
		},
	}
	result := diffPoliciesStub(t, cfg, nil)
	assertChangeKinds(t, result, engine.ChangeCreate, engine.ChangeCreate)
}

func TestDiff_PolicyNoChange(t *testing.T) {
	dom := "example.com"
	current := []api.TokenPolicy{
		{ID: "p1", Domain: nil, Subname: nil, Type: nil, PermWrite: false},
		{ID: "p2", Domain: &dom, Subname: nil, Type: nil, PermWrite: true},
	}
	cfg := config.TokenPolicies{
		TokenID: "t1",
		Policies: []config.Policy{
			{Domain: nil, Subname: nil, Type: nil, PermWrite: false},
			{Domain: &dom, Subname: nil, Type: nil, PermWrite: true},
		},
	}
	result := diffPoliciesStub(t, cfg, current)
	if len(result) != 0 {
		t.Fatalf("want 0 diffs when state matches, got %d: %+v", len(result), result)
	}
}

func TestDiff_PolicyUpdate(t *testing.T) {
	dom := "example.com"
	current := []api.TokenPolicy{
		{ID: "p1", Domain: nil, Subname: nil, Type: nil, PermWrite: false},
		{ID: "p2", Domain: &dom, Subname: nil, Type: nil, PermWrite: false}, // differs
	}
	cfg := config.TokenPolicies{
		TokenID: "t1",
		Policies: []config.Policy{
			{Domain: nil, Subname: nil, Type: nil, PermWrite: false},
			{Domain: &dom, Subname: nil, Type: nil, PermWrite: true},
		},
	}
	result := diffPoliciesStub(t, cfg, current)
	assertChangeKinds(t, result, engine.ChangeUpdate)
	if result[0].CurrentID != "p2" {
		t.Errorf("want CurrentID p2, got %s", result[0].CurrentID)
	}
}

func TestDiff_PolicyDelete(t *testing.T) {
	dom := "example.com"
	current := []api.TokenPolicy{
		{ID: "p1", Domain: nil, Subname: nil, Type: nil, PermWrite: false},
		{ID: "p2", Domain: &dom, Subname: nil, Type: nil, PermWrite: true}, // not in desired
	}
	cfg := config.TokenPolicies{
		TokenID: "t1",
		Policies: []config.Policy{
			{Domain: nil, Subname: nil, Type: nil, PermWrite: false},
		},
	}
	result := diffPoliciesStub(t, cfg, current)
	assertChangeKinds(t, result, engine.ChangeDelete)
	if result[0].CurrentID != "p2" {
		t.Errorf("want CurrentID p2, got %s", result[0].CurrentID)
	}
}

// ---- rrset diffs -----------------------------------------------------------

func TestDiff_RRsetCreate(t *testing.T) {
	cfg := config.Domain{
		Name: "example.com",
		RRsets: []config.RRset{
			{Subname: "www", Type: "A", Records: []string{"1.2.3.4"}},
		},
	}
	result := diffRRsetsStub(t, cfg, nil)
	if len(result) != 1 || result[0].Kind != engine.ChangeCreate {
		t.Fatalf("want 1 ChangeCreate, got %+v", result)
	}
}

func TestDiff_RRsetNoChange(t *testing.T) {
	current := []api.RRset{
		{Subname: "www", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
	}
	cfg := config.Domain{
		Name: "example.com",
		RRsets: []config.RRset{
			{Subname: "www", Type: "A", Records: []string{"1.2.3.4"}}, // TTL omitted → 3600
		},
	}
	result := diffRRsetsStub(t, cfg, current)
	if len(result) != 0 {
		t.Fatalf("want 0 diffs, got %+v", result)
	}
}

func TestDiff_RRsetUpdateTTL(t *testing.T) {
	current := []api.RRset{
		{Subname: "www", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
	}
	cfg := config.Domain{
		Name: "example.com",
		RRsets: []config.RRset{
			{Subname: "www", Type: "A", TTL: 300, Records: []string{"1.2.3.4"}},
		},
	}
	result := diffRRsetsStub(t, cfg, current)
	if len(result) != 1 || result[0].Kind != engine.ChangeUpdate {
		t.Fatalf("want 1 ChangeUpdate, got %+v", result)
	}
	if result[0].Diffs[0].Field != "ttl" {
		t.Errorf("want diff on ttl, got %s", result[0].Diffs[0].Field)
	}
}

func TestDiff_RRsetUpdateRecords(t *testing.T) {
	current := []api.RRset{
		{Subname: "", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
	}
	cfg := config.Domain{
		Name: "example.com",
		RRsets: []config.RRset{
			{Subname: "", Type: "A", Records: []string{"5.6.7.8"}},
		},
	}
	result := diffRRsetsStub(t, cfg, current)
	if len(result) != 1 || result[0].Kind != engine.ChangeUpdate {
		t.Fatalf("want 1 ChangeUpdate, got %+v", result)
	}
	if result[0].Diffs[0].Field != "records" {
		t.Errorf("want diff on records, got %s", result[0].Diffs[0].Field)
	}
}

func TestDiff_RRsetDelete(t *testing.T) {
	current := []api.RRset{
		{Subname: "www", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
		{Subname: "old", Type: "CNAME", TTL: 3600, Records: []string{"example.com."}},
	}
	cfg := config.Domain{
		Name: "example.com",
		RRsets: []config.RRset{
			{Subname: "www", Type: "A", Records: []string{"1.2.3.4"}},
		},
	}
	result := diffRRsetsStub(t, cfg, current)
	if len(result) != 1 || result[0].Kind != engine.ChangeDelete {
		t.Fatalf("want 1 ChangeDelete, got %+v", result)
	}
	if result[0].Subname != "old" || result[0].Type != "CNAME" {
		t.Errorf("want old/CNAME deleted, got %s/%s", result[0].Subname, result[0].Type)
	}
}

func TestDiff_RRsetIgnoresNS(t *testing.T) {
	current := []api.RRset{
		{Subname: "", Type: "NS", TTL: 3600, Records: []string{"ns1.desec.io."}},
		{Subname: "", Type: "A", TTL: 3600, Records: []string{"1.2.3.4"}},
	}
	cfg := config.Domain{
		Name: "example.com",
		RRsets: []config.RRset{
			{Subname: "", Type: "A", Records: []string{"1.2.3.4"}},
		},
	}
	result := diffRRsetsStub(t, cfg, current)
	if len(result) != 0 {
		t.Fatalf("NS record should be ignored, got diffs: %+v", result)
	}
}

func TestDiff_RRsetIgnoresSOA(t *testing.T) {
	current := []api.RRset{
		{Subname: "", Type: "SOA", TTL: 3600, Records: []string{"ns1. admin. 1 2 3 4 5"}},
	}
	cfg := config.Domain{Name: "example.com", RRsets: nil}
	result := diffRRsetsStub(t, cfg, current)
	if len(result) != 0 {
		t.Fatalf("SOA record should be ignored, got diffs: %+v", result)
	}
}

func TestDiff_RRsetRecordsOrderIndependent(t *testing.T) {
	current := []api.RRset{
		{Subname: "", Type: "A", TTL: 3600, Records: []string{"2.2.2.2", "1.1.1.1"}},
	}
	cfg := config.Domain{
		Name: "example.com",
		RRsets: []config.RRset{
			{Subname: "", Type: "A", Records: []string{"1.1.1.1", "2.2.2.2"}},
		},
	}
	result := diffRRsetsStub(t, cfg, current)
	if len(result) != 0 {
		t.Fatalf("record order should not matter, got diffs: %+v", result)
	}
}

// ---- helpers ---------------------------------------------------------------

// diffPoliciesStub runs Diff for a single token policy group using a stub that
// returns the given current policies.
func diffPoliciesStub(t *testing.T, desired config.TokenPolicies, current []api.TokenPolicy) []engine.PolicyDiff {
	t.Helper()
	stub := &stubClient{
		listPoliciesFn: func(string) ([]api.TokenPolicy, error) { return current, nil },
	}
	cfg := &config.Config{TokenPolicies: []config.TokenPolicies{desired}}
	result, err := engine.Diff(cfg, stub)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if len(result.TokenPolicies) == 0 {
		return nil
	}
	return result.TokenPolicies[0].Policies
}

// diffRRsetsStub runs Diff for a single domain using a stub that returns the
// given current rrsets.
func diffRRsetsStub(t *testing.T, desired config.Domain, current []api.RRset) []engine.RRsetDiff {
	t.Helper()
	stub := &stubClient{
		listRRsetsFn: func(string) ([]api.RRset, error) { return current, nil },
	}
	cfg := &config.Config{Domains: []config.Domain{desired}}
	result, err := engine.Diff(cfg, stub)
	if err != nil {
		t.Fatalf("Diff error: %v", err)
	}
	if len(result.Domains) == 0 {
		return nil
	}
	return result.Domains[0].Changes
}

func assertChangeKinds(t *testing.T, diffs []engine.PolicyDiff, want ...engine.ChangeKind) {
	t.Helper()
	if len(diffs) != len(want) {
		t.Fatalf("want %d diffs, got %d: %+v", len(want), len(diffs), diffs)
	}
	for i, w := range want {
		if diffs[i].Kind != w {
			t.Errorf("diff[%d]: want kind %d, got %d", i, w, diffs[i].Kind)
		}
	}
}
