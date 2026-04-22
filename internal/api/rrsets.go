package api

import "fmt"

// ListRRsets returns all RRsets for the given domain name.
func (c *Client) ListRRsets(domain string) ([]RRset, error) {
	var rrsets []RRset
	path := "/domains/" + domain + "/rrsets/"
	if err := c.listAll(path, &rrsets); err != nil {
		return nil, fmt.Errorf("listing rrsets for %s: %w", domain, err)
	}
	return rrsets, nil
}

// BulkPatchRRsets atomically applies a set of RRset changes to a domain via a
// single PATCH request. Entries with an empty Records slice are deleted; all
// others are created or updated. Using a single request counts against
// dns_api_per_domain_expensive once instead of once per RRset.
func (c *Client) BulkPatchRRsets(domain string, items []RRsetWriteFields) error {
	path := "/domains/" + domain + "/rrsets/"
	code, err := c.request("PATCH", path, items, nil)
	if err != nil {
		return fmt.Errorf("bulk patching rrsets for %s: %w", domain, err)
	}
	// 200 (changes applied) and 204 (no effective change) are both success.
	if code != 200 && code != 204 {
		return fmt.Errorf("bulk patching rrsets for %s: unexpected status %d", domain, code)
	}
	return nil
}
