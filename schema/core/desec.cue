package core

// Desec holds everything desync manages in a deSEC account: the record sets
// of all zones and the token policies.
#Desec: {
	zones: #Zones

	// policies by token id
	tokenPolicies: [TokenId=string]: [...#TokenPolicyRecord]
}

// Zones holds the record sets of all zones by zone, subname and type:
//
//	zones: "example.org": www: A: records: ["192.0.2.1"]
//
// Each record set has exactly one place, so two definitions of the same record
// set are unified: equal records merge, different records fail evaluation.
//
// The zone apex has the subname "".
#Zones: [Zone=string]: [Subname=string]: close({[=~"^[A-Z][A-Z0-9]*$"]: #RRSet})

// A record set of one subname and type
#RRSet: {
	records!: [...string]
}
