// Package engine implements the plan/apply lifecycle: it computes the diff
// between the desired state (config file) and the current remote state (deSEC
// API), prints a human-readable plan, and executes the necessary API calls.
package engine

import (
	"fmt"
	"sort"

	"codeberg.org/xchangeee/desync/internal/api"
	"codeberg.org/xchangeee/desync/internal/config"
)

// managedByDesec lists record types controlled exclusively by deSEC that
// desync must never attempt to create, update, or delete.
var managedByDesec = map[string]bool{
	"NS":  true,
	"SOA": true,
}

// ChangeKind classifies a planned change.
type ChangeKind int

const (
	ChangeCreate ChangeKind = iota
	ChangeUpdate
	ChangeDelete
)

// FieldDiff records a single field that differs between desired and current state.
type FieldDiff struct {
	Field   string
	Current string
	Desired string
}

// PolicyDiff describes a planned change to a token scoping policy.
type PolicyDiff struct {
	Kind      ChangeKind
	CurrentID string  // empty when Kind == ChangeCreate
	Domain    *string // identity key
	Subname   *string // identity key
	Type      *string // identity key
	Diffs     []FieldDiff
}

// TokenPoliciesDiff describes all planned policy changes for one token.
type TokenPoliciesDiff struct {
	TokenID  string
	Policies []PolicyDiff
}

// RRsetDiff describes a planned change to a single RRset within a domain.
type RRsetDiff struct {
	Kind    ChangeKind
	Subname string
	Type    string
	Diffs   []FieldDiff
	Desired *config.RRset // nil when Kind == ChangeDelete
}

// DomainDiff groups all RRset changes for one domain.
type DomainDiff struct {
	Domain  string
	Changes []RRsetDiff
}

// DiffResult is the complete plan: what would change if Apply were run.
type DiffResult struct {
	TokenPolicies []TokenPoliciesDiff
	Domains       []DomainDiff
}

// Counts returns the total number of creates, updates, and deletes.
func (d *DiffResult) Counts() (create, update, del int) {
	count := func(k ChangeKind) {
		switch k {
		case ChangeCreate:
			create++
		case ChangeUpdate:
			update++
		case ChangeDelete:
			del++
		}
	}
	for _, tp := range d.TokenPolicies {
		for _, p := range tp.Policies {
			count(p.Kind)
		}
	}
	for _, dd := range d.Domains {
		for _, r := range dd.Changes {
			count(r.Kind)
		}
	}
	return
}

// Diff fetches the current remote state from the API and computes the full
// diff against the desired state in cfg. It does not modify anything.
func Diff(cfg *config.Config, client *api.Client) (*DiffResult, error) {
	result := &DiffResult{}

	policyDiffs, err := diffTokenPolicies(cfg.TokenPolicies, client)
	if err != nil {
		return nil, err
	}
	result.TokenPolicies = policyDiffs

	domainDiffs, err := diffDomains(cfg.Domains, client)
	if err != nil {
		return nil, err
	}
	result.Domains = domainDiffs

	return result, nil
}

// ---- token policies --------------------------------------------------------

func diffTokenPolicies(desired []config.TokenPolicies, client *api.Client) ([]TokenPoliciesDiff, error) {
	var diffs []TokenPoliciesDiff
	for _, tp := range desired {
		current, err := client.ListPolicies(tp.TokenID)
		if err != nil {
			return nil, fmt.Errorf("listing policies for token %s: %w", tp.TokenID, err)
		}
		diffs = append(diffs, TokenPoliciesDiff{
			TokenID:  tp.TokenID,
			Policies: planPolicies(tp.Policies, current),
		})
	}
	return diffs, nil
}

func planPolicies(desired []config.Policy, current []api.TokenPolicy) []PolicyDiff {
	type policyKey struct{ domain, subname, ptype string }
	key := func(d, s, t *string) policyKey {
		return policyKey{ptrStr(d), ptrStr(s), ptrStr(t)}
	}

	curByKey := make(map[policyKey]api.TokenPolicy)
	for _, p := range current {
		curByKey[key(p.Domain, p.Subname, p.Type)] = p
	}

	desiredKeys := make(map[policyKey]bool)
	var diffs []PolicyDiff

	for _, dp := range desired {
		k := key(dp.Domain, dp.Subname, dp.Type)
		desiredKeys[k] = true

		if cur, ok := curByKey[k]; !ok {
			diffs = append(diffs, PolicyDiff{
				Kind:    ChangeCreate,
				Domain:  dp.Domain,
				Subname: dp.Subname,
				Type:    dp.Type,
				Diffs:   []FieldDiff{{Field: "perm_write", Current: "", Desired: fmt.Sprintf("%v", dp.PermWrite)}},
			})
		} else if cur.PermWrite != dp.PermWrite {
			diffs = append(diffs, PolicyDiff{
				Kind:      ChangeUpdate,
				CurrentID: cur.ID,
				Domain:    dp.Domain,
				Subname:   dp.Subname,
				Type:      dp.Type,
				Diffs:     []FieldDiff{{Field: "perm_write", Current: fmt.Sprintf("%v", cur.PermWrite), Desired: fmt.Sprintf("%v", dp.PermWrite)}},
			})
		}
	}

	for _, cp := range current {
		if !desiredKeys[key(cp.Domain, cp.Subname, cp.Type)] {
			diffs = append(diffs, PolicyDiff{
				Kind:      ChangeDelete,
				CurrentID: cp.ID,
				Domain:    cp.Domain,
				Subname:   cp.Subname,
				Type:      cp.Type,
			})
		}
	}

	return diffs
}

// ---- domains / rrsets ------------------------------------------------------

func diffDomains(desired []config.Domain, client *api.Client) ([]DomainDiff, error) {
	var results []DomainDiff

	for _, d := range desired {
		current, err := client.ListRRsets(d.Name)
		if err != nil {
			return nil, err
		}

		type rrKey struct{ subname, rrtype string }
		curByKey := make(map[rrKey]api.RRset)
		for _, r := range current {
			if !managedByDesec[r.Type] {
				curByKey[rrKey{r.Subname, r.Type}] = r
			}
		}

		desiredKeys := make(map[rrKey]bool)
		var changes []RRsetDiff

		for i := range d.RRsets {
			dr := &d.RRsets[i]
			k := rrKey{dr.Subname, dr.Type}
			desiredKeys[k] = true
			wantTTL := dr.EffectiveTTL()

			if cur, ok := curByKey[k]; !ok {
				changes = append(changes, RRsetDiff{
					Kind:    ChangeCreate,
					Subname: dr.Subname,
					Type:    dr.Type,
					Desired: dr,
				})
			} else {
				var diffs []FieldDiff
				if cur.TTL != wantTTL {
					diffs = append(diffs, FieldDiff{
						Field:   "ttl",
						Current: fmt.Sprintf("%d", cur.TTL),
						Desired: fmt.Sprintf("%d", wantTTL),
					})
				}
				if !strSlicesEqual(cur.Records, dr.Records) {
					diffs = append(diffs, FieldDiff{
						Field:   "records",
						Current: joinStrs(cur.Records),
						Desired: joinStrs(dr.Records),
					})
				}
				if len(diffs) > 0 {
					changes = append(changes, RRsetDiff{
						Kind:    ChangeUpdate,
						Subname: dr.Subname,
						Type:    dr.Type,
						Diffs:   diffs,
						Desired: dr,
					})
				}
			}
		}

		for _, cr := range current {
			if !desiredKeys[rrKey{cr.Subname, cr.Type}] {
				changes = append(changes, RRsetDiff{
					Kind:    ChangeDelete,
					Subname: cr.Subname,
					Type:    cr.Type,
				})
			}
		}

		if len(changes) > 0 {
			results = append(results, DomainDiff{Domain: d.Name, Changes: changes})
		}
	}

	return results, nil
}

// ---- helpers ---------------------------------------------------------------

func ptrStr(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

func strSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}

func joinStrs(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	cp := append([]string(nil), ss...)
	sort.Strings(cp)
	out := "["
	for i, s := range cp {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out + "]"
}
