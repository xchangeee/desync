// Input config schema for the desync tool
package core

// Represents a managed desec account
#Desync: {
	// All domains under management
	domains!: [...#Domain]

	// All token policies under management
	tokenPolicies!: [...#TokenPolicy]
}

// A managed domain
#Domain: {
	// The domain name
	name!: string

	// All record sets
	rrsets!: [...#DomainRecordSet]
}

// A domain record set
#DomainRecordSet: {
	// Subdomain name
	subname!: string

	// Record type
	type!: string

	// Records
	records!: [...string]
}

// Token policies control fine-grained access to DNS data
#TokenPolicy: {
	// the token uuid
	tokenId!: string

	// policy records for this token
	policies!: [...#TokenPolicyRecord]
}

// A token policy record
#TokenPolicyRecord: {
	// Domain target
	domain!: string | null

	// Subdomain target
	subname!: string | null

	// Type target
	type!: string | null

	// read or write
	permWrite!: bool
}
