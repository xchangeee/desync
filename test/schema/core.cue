// Tests for the zone structure of core.#Desec and the config schema
// core.#Desync: how record sets merge, and which values the structure rejects.
package schema

import "github.com/xchangeee/desync/schema/core@v0"

// Zones

// two definitions of the same record set with equal records merge
tests: zonesMerge: {
	got: core.#Zones & {
		"example.org": www: A: records: ["192.0.2.1"]
		"example.org": www: A: records: ["192.0.2.1"]
		"example.org": "": MX: records: ["10 mx.example.org."]
	}
	want: "example.org": {
		www: A: records: ["192.0.2.1"]
		"": MX: records: ["10 mx.example.org."]
	}
}

// two definitions of the same record set with different records fail
rejects: zonesConflict: (core.#Zones & {
	"example.org": www: A: records: ["192.0.2.1"]
	"example.org": www: A: records: ["192.0.2.2"]
}) == _|_

// types are upper case
rejects: zonesLowerCaseType: (core.#Zones & {"example.org": www: a: records: ["192.0.2.1"]}) == _|_

// a record set holds nothing but its records
rejects: zonesUnknownField: (core.#Zones & {"example.org": www: A: {records: ["192.0.2.1"], ttl: 60}}) == _|_

// Desec

tests: desec: {
	got: core.#Desec & {
		zones: "example.org": www: A: records: ["192.0.2.1"]
		tokenPolicies: "0f1e2d3c-0000-0000-0000-000000000000": [
			{domain: null, subname: null, type: null, permWrite: false},
			{domain: "example.org", subname: "www", type: "A", permWrite: true},
		]
	}
	want: {
		zones: "example.org": www: A: records: ["192.0.2.1"]
		tokenPolicies: "0f1e2d3c-0000-0000-0000-000000000000": [
			{domain: null, subname: null, type: null, permWrite: false},
			{domain: "example.org", subname: "www", type: "A", permWrite: true},
		]
	}
}

rejects: desecPolicyPermNotBool: (core.#Desec & {
	zones: {}
	tokenPolicies: "0f1e2d3c-0000-0000-0000-000000000000": [{domain: null, subname: null, type: null, permWrite: "yes"}]
}) == _|_

// Desync

rejects: desyncUnknownField: (core.#Desync & {domains: [], tokenPolicies: [], ttl: 60}) == _|_
