package engine

import (
	"fmt"
	"sort"

	"codeberg.org/xchangeee/desync/internal/api"
	"codeberg.org/xchangeee/desync/internal/config"
)

// Apply executes the changes described in d against the API.
// Resources are processed in dependency order:
//  1. Token policies (independent of RRsets)
//  2. RRsets (bulk-patched per domain to minimise rate-limit consumption)
func Apply(d *DiffResult, cfg *config.Config, client *api.Client) error {
	// Build a lookup of desired policies by token ID.
	desiredByToken := make(map[string][]config.Policy)
	for _, tp := range cfg.TokenPolicies {
		desiredByToken[tp.TokenID] = tp.Policies
	}

	for _, td := range d.TokenPolicies {
		if len(td.Policies) == 0 {
			continue
		}
		if err := applyPolicies(td.TokenID, td.Policies, desiredByToken[td.TokenID], client); err != nil {
			return err
		}
	}

	// Build a lookup of desired RRsets by domain.
	desiredRRsets := make(map[string][]config.RRset)
	for _, r := range cfg.RRsets {
		desiredRRsets[r.Domain] = append(desiredRRsets[r.Domain], r)
	}

	for _, dr := range d.RRsets {
		if err := applyRRsets(dr, client); err != nil {
			return err
		}
	}

	return nil
}

// applyPolicies reconciles policies on a token.
// The deSEC API requires the default policy (all-null key) to be created
// before any specific policy, and deleted last. We sort accordingly.
func applyPolicies(tokenID string, diffs []PolicyDiff, desired []config.Policy, client *api.Client) error {
	type policyKey struct{ domain, subname, ptype string }
	key := func(d, s, t *string) policyKey { return policyKey{ptrStr(d), ptrStr(s), ptrStr(t)} }

	desiredByKey := make(map[policyKey]config.Policy)
	for _, p := range desired {
		desiredByKey[key(p.Domain, p.Subname, p.Type)] = p
	}

	// Sort: creates before updates before deletes; within creates, default
	// policy (all-null) comes first; within deletes, default policy comes last.
	isDefault := func(p PolicyDiff) bool {
		return p.Domain == nil && p.Subname == nil && p.Type == nil
	}
	kindRank := func(k ChangeKind) int {
		switch k {
		case ChangeCreate:
			return 0
		case ChangeUpdate:
			return 1
		case ChangeDelete:
			return 2
		}
		return 3
	}
	sort.SliceStable(diffs, func(i, j int) bool {
		pi, pj := diffs[i], diffs[j]
		if pi.Kind != pj.Kind {
			return kindRank(pi.Kind) < kindRank(pj.Kind)
		}
		// Same kind: default policy first on create, last on delete.
		if pi.Kind == ChangeCreate {
			return isDefault(pi) && !isDefault(pj)
		}
		if pi.Kind == ChangeDelete {
			return !isDefault(pi) && isDefault(pj)
		}
		return false
	})

	fmt.Printf("Reconciling policies for token %s...\n", tokenID)
	for _, pd := range diffs {
		k := key(pd.Domain, pd.Subname, pd.Type)
		label := fmt.Sprintf("{domain=%s subname=%s type=%s}", ptrStr(pd.Domain), ptrStr(pd.Subname), ptrStr(pd.Type))
		fields := api.TokenPolicyWriteFields{
			Domain:  pd.Domain,
			Subname: pd.Subname,
			Type:    pd.Type,
		}
		if dp, ok := desiredByKey[k]; ok {
			fields.PermWrite = dp.PermWrite
		}

		switch pd.Kind {
		case ChangeCreate:
			fmt.Printf("  + create policy %s\n", label)
			if _, err := client.CreatePolicy(tokenID, fields); err != nil {
				return err
			}
		case ChangeUpdate:
			fmt.Printf("  ~ update policy %s\n", label)
			if err := client.UpdatePolicy(tokenID, pd.CurrentID, fields); err != nil {
				return err
			}
		case ChangeDelete:
			fmt.Printf("  - delete policy %s\n", label)
			if err := client.DeletePolicy(tokenID, pd.CurrentID); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyRRsets sends a single bulk PATCH for all RRset changes on one domain.
// Deletions are expressed as entries with an empty records list, per the API spec.
func applyRRsets(dr DomainRRsetDiff, client *api.Client) error {
	var items []api.RRsetWriteFields
	creates, updates, deletes := 0, 0, 0

	for _, ch := range dr.Changes {
		switch ch.Kind {
		case ChangeCreate:
			creates++
			items = append(items, api.RRsetWriteFields{
				Subname: ch.Desired.Subname,
				Type:    ch.Desired.Type,
				TTL:     ch.Desired.TTL,
				Records: ch.Desired.Records,
			})
		case ChangeUpdate:
			updates++
			items = append(items, api.RRsetWriteFields{
				Subname: ch.Desired.Subname,
				Type:    ch.Desired.Type,
				TTL:     ch.Desired.TTL,
				Records: ch.Desired.Records,
			})
		case ChangeDelete:
			deletes++
			// Empty records array signals deletion in a bulk PATCH.
			items = append(items, api.RRsetWriteFields{
				Subname: ch.Subname,
				Type:    ch.Type,
				Records: []string{},
			})
		}
	}

	if len(items) == 0 {
		return nil
	}

	fmt.Printf("Applying RRsets for %q (%d create, %d update, %d delete)...\n",
		dr.Domain, creates, updates, deletes)
	if err := client.BulkPatchRRsets(dr.Domain, items); err != nil {
		return err
	}
	fmt.Printf("  Done.\n")
	return nil
}
