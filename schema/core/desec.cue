package core

// domains in cue are managed in the desec struct
#Desec: {
	// Domains by name
	domains: [string]: [...#DomainRecordSet]

	// Token Policies by uuid
	tokenPolicies: [string]: [...#TokenPolicyRecord]
}
