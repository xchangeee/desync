package engine

import (
	"fmt"

	"codeberg.org/xchangeee/desync/internal/api"
	"codeberg.org/xchangeee/desync/internal/config"
)

// Apply executes the changes described in d against the API.
// Resources are processed in dependency order:
//  1. Token policies
//  2. RRsets (bulk-patched per domain to minimise rate-limit consumption)
func Apply(d *DiffResult, cfg *config.Config, client Client) error {
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

	desiredRRsets := make(map[string][]config.RRset)
	for _, dom := range cfg.Domains {
		desiredRRsets[dom.Name] = dom.RRsets
	}

	for _, dd := range d.Domains {
		if err := applyRRsets(dd, client); err != nil {
			return err
		}
	}

	return nil
}

// applyPolicies reconciles policies on a token.
// The deSEC API requires the default policy (all-null key) to be created
// before any specific policy, and deleted last. We sort accordingly.
func applyPolicies(tokenID string, diffs []PolicyDiff, desired []config.Policy, client Client) error {
	type policyKey struct{ domain, subname, ptype string }
	key := func(d, s, t *string) policyKey { return policyKey{ptrStr(d), ptrStr(s), ptrStr(t)} }

	desiredByKey := make(map[policyKey]config.Policy)
	for _, p := range desired {
		desiredByKey[key(p.Domain, p.Subname, p.Type)] = p
	}

	isDefault := func(p PolicyDiff) bool {
		return p.Domain == nil && p.Subname == nil && p.Type == nil
	}

	// The deSEC API enforces strict ordering:
	//   - the default policy must be created before any specific policies
	//   - all specific policies must be deleted before the default policy
	// Build an explicitly ordered slice instead of relying on a sort comparator.
	var (
		defaultCreates  []PolicyDiff
		specificCreates []PolicyDiff
		updates         []PolicyDiff
		specificDeletes []PolicyDiff
		defaultDeletes  []PolicyDiff
	)
	for _, pd := range diffs {
		switch pd.Kind {
		case ChangeCreate:
			if isDefault(pd) {
				defaultCreates = append(defaultCreates, pd)
			} else {
				specificCreates = append(specificCreates, pd)
			}
		case ChangeUpdate:
			updates = append(updates, pd)
		case ChangeDelete:
			if isDefault(pd) {
				defaultDeletes = append(defaultDeletes, pd)
			} else {
				specificDeletes = append(specificDeletes, pd)
			}
		}
	}
	ordered := make([]PolicyDiff, 0, len(diffs))
	ordered = append(ordered, defaultCreates...)
	ordered = append(ordered, specificCreates...)
	ordered = append(ordered, updates...)
	ordered = append(ordered, specificDeletes...)
	ordered = append(ordered, defaultDeletes...)

	fmt.Printf("Reconciling policies for token %s...\n", tokenID)
	for _, pd := range ordered {
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
func applyRRsets(dd DomainDiff, client Client) error {
	var items []api.RRsetWriteFields
	creates, updates, deletes := 0, 0, 0

	for _, ch := range dd.Changes {
		switch ch.Kind {
		case ChangeCreate, ChangeUpdate:
			creates++
			if ch.Kind == ChangeUpdate {
				creates--
				updates++
			}
			items = append(items, api.RRsetWriteFields{
				Subname: ch.Desired.Subname,
				Type:    ch.Desired.Type,
				TTL:     ch.Desired.EffectiveTTL(),
				Records: ch.Desired.Records,
			})
		case ChangeDelete:
			deletes++
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
		dd.Domain, creates, updates, deletes)
	if err := client.BulkPatchRRsets(dd.Domain, items); err != nil {
		return err
	}
	fmt.Printf("  Done.\n")
	return nil
}
