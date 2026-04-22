package api

import "fmt"

// ListDomains returns all domains in the authenticated account.
func (c *Client) ListDomains() ([]Domain, error) {
	var domains []Domain
	if err := c.listAll("/domains/", &domains); err != nil {
		return nil, fmt.Errorf("listing domains: %w", err)
	}
	return domains, nil
}

// CreateDomain creates a new domain. zonefile may be empty.
func (c *Client) CreateDomain(name, zonefile string) (*Domain, error) {
	body := map[string]string{"name": name}
	if zonefile != "" {
		body["zonefile"] = zonefile
	}
	var d Domain
	if _, err := c.request("POST", "/domains/", body, &d); err != nil {
		return nil, fmt.Errorf("creating domain %q: %w", name, err)
	}
	return &d, nil
}
