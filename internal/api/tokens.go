package api

import "fmt"

// ListTokens returns all tokens visible to the authenticated user.
func (c *Client) ListTokens() ([]Token, error) {
	var tokens []Token
	if err := c.listAll("/auth/tokens/", &tokens); err != nil {
		return nil, fmt.Errorf("listing tokens: %w", err)
	}
	return tokens, nil
}

// CreateToken creates a new token and returns it including the one-time secret value.
func (c *Client) CreateToken(fields TokenWriteFields) (*Token, error) {
	var t Token
	if _, err := c.request("POST", "/auth/tokens/", fields, &t); err != nil {
		return nil, fmt.Errorf("creating token %q: %w", fields.Name, err)
	}
	return &t, nil
}

// UpdateToken patches the mutable fields of an existing token by its ID.
func (c *Client) UpdateToken(id string, fields TokenWriteFields) error {
	if _, err := c.request("PATCH", "/auth/tokens/"+id+"/", fields, nil); err != nil {
		return fmt.Errorf("updating token %s: %w", id, err)
	}
	return nil
}

// ListPolicies returns all scoping policies for the token identified by tokenID.
func (c *Client) ListPolicies(tokenID string) ([]TokenPolicy, error) {
	var policies []TokenPolicy
	path := "/auth/tokens/" + tokenID + "/policies/rrsets/"
	if err := c.listAll(path, &policies); err != nil {
		return nil, fmt.Errorf("listing policies for token %s: %w", tokenID, err)
	}
	return policies, nil
}

// CreatePolicy creates a new scoping policy on the given token.
func (c *Client) CreatePolicy(tokenID string, fields TokenPolicyWriteFields) (*TokenPolicy, error) {
	var p TokenPolicy
	path := "/auth/tokens/" + tokenID + "/policies/rrsets/"
	if _, err := c.request("POST", path, fields, &p); err != nil {
		return nil, fmt.Errorf("creating policy on token %s: %w", tokenID, err)
	}
	return &p, nil
}

// UpdatePolicy patches an existing policy identified by policyID on the given token.
func (c *Client) UpdatePolicy(tokenID, policyID string, fields TokenPolicyWriteFields) error {
	path := "/auth/tokens/" + tokenID + "/policies/rrsets/" + policyID + "/"
	if _, err := c.request("PATCH", path, fields, nil); err != nil {
		return fmt.Errorf("updating policy %s on token %s: %w", policyID, tokenID, err)
	}
	return nil
}

// DeletePolicy removes a scoping policy from the given token.
// Per API rules the default policy (domain/subname/type all null) must be
// deleted last; callers are responsible for correct ordering.
func (c *Client) DeletePolicy(tokenID, policyID string) error {
	path := "/auth/tokens/" + tokenID + "/policies/rrsets/" + policyID + "/"
	if _, err := c.request("DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("deleting policy %s on token %s: %w", policyID, tokenID, err)
	}
	return nil
}
