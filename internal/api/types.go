package api

// Token is the API representation of a deSEC authentication token.
// The "token" field (the secret value) is only populated on creation.
type Token struct {
	ID               string   `json:"id"`
	Created          string   `json:"created"`
	LastUsed         *string  `json:"last_used"`
	Owner            string   `json:"owner"`
	UserOverride     *string  `json:"user_override"`
	MFA              *bool    `json:"mfa"`
	MaxAge           *string  `json:"max_age"`
	MaxUnusedPeriod  *string  `json:"max_unused_period"`
	Name             string   `json:"name"`
	PermCreateDomain bool     `json:"perm_create_domain"`
	PermDeleteDomain bool     `json:"perm_delete_domain"`
	PermManageTokens bool     `json:"perm_manage_tokens"`
	AllowedSubnets   []string `json:"allowed_subnets"`
	AutoPolicy       bool     `json:"auto_policy"`
	IsValid          bool     `json:"is_valid"`
	Secret           string   `json:"token,omitempty"` // only present on creation
}

// TokenWriteFields contains the subset of Token fields that can be set via
// POST or PATCH on the tokens endpoint.
type TokenWriteFields struct {
	Name             string   `json:"name"`
	PermCreateDomain bool     `json:"perm_create_domain"`
	PermDeleteDomain bool     `json:"perm_delete_domain"`
	PermManageTokens bool     `json:"perm_manage_tokens"`
	AllowedSubnets   []string `json:"allowed_subnets"`
	MaxAge           *string  `json:"max_age"`
	MaxUnusedPeriod  *string  `json:"max_unused_period"`
	AutoPolicy       bool     `json:"auto_policy"`
}

// TokenPolicy is the API representation of a token scoping policy.
// The combination of (Domain, Subname, Type) identifies the policy uniquely
// within a token; a nil value in any of those fields means "wildcard / default".
type TokenPolicy struct {
	ID        string  `json:"id"`
	Domain    *string `json:"domain"`
	Subname   *string `json:"subname"`
	Type      *string `json:"type"`
	PermWrite bool    `json:"perm_write"`
}

// TokenPolicyWriteFields contains the mutable fields of a TokenPolicy.
type TokenPolicyWriteFields struct {
	Domain    *string `json:"domain"`
	Subname   *string `json:"subname"`
	Type      *string `json:"type"`
	PermWrite bool    `json:"perm_write"`
}

// Domain is the API representation of a deSEC DNS zone.
type Domain struct {
	Created    string `json:"created"`
	MinimumTTL int    `json:"minimum_ttl"`
	Name       string `json:"name"`
	Published  string `json:"published"`
	Touched    string `json:"touched"`
}

// RRset is the API representation of a DNS Resource Record Set.
type RRset struct {
	Created string   `json:"created"`
	Domain  string   `json:"domain"`
	Subname string   `json:"subname"`
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Records []string `json:"records"`
	TTL     int      `json:"ttl"`
	Touched string   `json:"touched"`
}

// RRsetWriteFields is the payload used for creating or modifying an RRset.
// An empty Records slice signals deletion in bulk PATCH operations.
type RRsetWriteFields struct {
	Subname string   `json:"subname"`
	Type    string   `json:"type"`
	TTL     int      `json:"ttl,omitempty"`
	Records []string `json:"records"`
}
