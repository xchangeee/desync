package engine

import (
	"codeberg.org/xchangeee/desync/internal/api"
)

// Client is the subset of the deSEC API used by the engine. The real
// *api.Client satisfies this interface; tests supply a stub implementation.
type Client interface {
	ListTokens() ([]api.Token, error)
	ListPolicies(tokenID string) ([]api.TokenPolicy, error)
	CreatePolicy(tokenID string, fields api.TokenPolicyWriteFields) (*api.TokenPolicy, error)
	UpdatePolicy(tokenID, policyID string, fields api.TokenPolicyWriteFields) error
	DeletePolicy(tokenID, policyID string) error
	ListDomains() ([]api.Domain, error)
	ListRRsets(domain string) ([]api.RRset, error)
	BulkPatchRRsets(domain string, items []api.RRsetWriteFields) error
	CreateToken(fields api.TokenWriteFields) (*api.Token, error)
}
