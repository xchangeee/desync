package engine_test

import (
	"github.com/xchangeee/desync/internal/api"
)

// stubClient implements engine.Client with function fields so each test can
// supply only the methods it needs; unset methods return zero values.
type stubClient struct {
	listTokensFn      func() ([]api.Token, error)
	listPoliciesFn    func(tokenID string) ([]api.TokenPolicy, error)
	createPolicyFn    func(tokenID string, f api.TokenPolicyWriteFields) (*api.TokenPolicy, error)
	updatePolicyFn    func(tokenID, policyID string, f api.TokenPolicyWriteFields) error
	deletePolicyFn    func(tokenID, policyID string) error
	listDomainsFn     func() ([]api.Domain, error)
	listRRsetsFn      func(domain string) ([]api.RRset, error)
	bulkPatchRRsetsFn func(domain string, items []api.RRsetWriteFields) error
	createTokenFn     func(f api.TokenWriteFields) (*api.Token, error)
}

func (s *stubClient) ListTokens() ([]api.Token, error) {
	if s.listTokensFn != nil {
		return s.listTokensFn()
	}
	return nil, nil
}
func (s *stubClient) ListPolicies(tokenID string) ([]api.TokenPolicy, error) {
	if s.listPoliciesFn != nil {
		return s.listPoliciesFn(tokenID)
	}
	return nil, nil
}
func (s *stubClient) CreatePolicy(tokenID string, f api.TokenPolicyWriteFields) (*api.TokenPolicy, error) {
	if s.createPolicyFn != nil {
		return s.createPolicyFn(tokenID, f)
	}
	return &api.TokenPolicy{}, nil
}
func (s *stubClient) UpdatePolicy(tokenID, policyID string, f api.TokenPolicyWriteFields) error {
	if s.updatePolicyFn != nil {
		return s.updatePolicyFn(tokenID, policyID, f)
	}
	return nil
}
func (s *stubClient) DeletePolicy(tokenID, policyID string) error {
	if s.deletePolicyFn != nil {
		return s.deletePolicyFn(tokenID, policyID)
	}
	return nil
}
func (s *stubClient) ListDomains() ([]api.Domain, error) {
	if s.listDomainsFn != nil {
		return s.listDomainsFn()
	}
	return nil, nil
}
func (s *stubClient) ListRRsets(domain string) ([]api.RRset, error) {
	if s.listRRsetsFn != nil {
		return s.listRRsetsFn(domain)
	}
	return nil, nil
}
func (s *stubClient) BulkPatchRRsets(domain string, items []api.RRsetWriteFields) error {
	if s.bulkPatchRRsetsFn != nil {
		return s.bulkPatchRRsetsFn(domain, items)
	}
	return nil
}
func (s *stubClient) CreateToken(f api.TokenWriteFields) (*api.Token, error) {
	if s.createTokenFn != nil {
		return s.createTokenFn(f)
	}
	return &api.Token{}, nil
}

// helpers

//go:fix inline
func strPtr(s string) *string { return new(s) }
