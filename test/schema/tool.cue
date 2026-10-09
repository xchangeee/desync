// Tests for the JSON rendering of core.#Desec by tool.#DesyncJsonFromZones.
//
// The expected values follow config.Config in internal/config, which reads the
// JSON in desync: they are what that JSON has to decode to. The JSON is decoded
// before comparing, so the field order of the output does not matter; the order
// of lists does.
package schema

import (
	"encoding/json"

	"github.com/xchangeee/desync/schema/core@v0"
	"github.com/xchangeee/desync/schema/record@v0"
	"github.com/xchangeee/desync/schema/tool@v0"
)

tests: jsonEmpty: {
	got: json.Unmarshal((tool.#DesyncJsonFromZones & {in: {zones: {}, tokenPolicies: {}}}).out)
	want: {domains: [], tokenPolicies: []}
}

// one record set per zone, subname and type, the zone apex as ""
tests: jsonZones: {
	got: json.Unmarshal((tool.#DesyncJsonFromZones & {in: {
		zones: {
			"example.org": "": MX: records: ["10 mx.example.org."]
			"example.org": www: {A: records: ["192.0.2.1"], AAAA: records: ["2001:db8::1"]}
			"example.net": "": A: records: ["192.0.2.2"]
		}
		tokenPolicies: {}
	}}).out)
	want: {
		domains: [
			{name: "example.org", rrsets: [
				{subname: "", type: "MX", records: ["10 mx.example.org."]},
				{subname: "www", type: "A", records: ["192.0.2.1"]},
				{subname: "www", type: "AAAA", records: ["2001:db8::1"]},
			]},
			{name: "example.net", rrsets: [
				{subname: "", type: "A", records: ["192.0.2.2"]},
			]},
		]
		tokenPolicies: []
	}
}

tests: jsonTokenPolicies: {
	got: json.Unmarshal((tool.#DesyncJsonFromZones & {in: {
		zones: {}
		tokenPolicies: "0f1e2d3c-0000-0000-0000-000000000000": [
			{domain: null, subname: null, type: null, permWrite: false},
			{domain: "example.org", subname: null, type: "TXT", permWrite: true},
		]
	}}).out)
	want: {
		domains: []
		tokenPolicies: [{
			tokenId: "0f1e2d3c-0000-0000-0000-000000000000"
			policies: [
				{domain: null, subname: null, type: null, permWrite: false},
				{domain: "example.org", subname: null, type: "TXT", permWrite: true},
			]
		}]
	}
}

// the output satisfies the config schema
tests: jsonSchema: {
	got: core.#Desync & json.Unmarshal((tool.#DesyncJsonFromZones & {in: jsonZonesFromBuilders}).out)
	want: json.Unmarshal((tool.#DesyncJsonFromZones & {in: jsonZonesFromBuilders}).out)
}

// end to end: a zone built from the record set builders, as a config file
// would do it
//
// The record sets of a zone are compared by "<subname> <type>" instead of as a
// list: their order follows the field order of the unified zone, which is not
// the order they are written in, and desync reconciles record sets by subname
// and type anyway.
tests: jsonBuilders: {
	got: {
		let Config = json.Unmarshal((tool.#DesyncJsonFromZones & {in: jsonZonesFromBuilders}).out)
		for d in Config.domains {
			(d.name): {for r in d.rrsets {"\(r.subname) \(r.type)": r.records}}
		}
	}
	want: "example.org": {
		" A": ["192.0.2.1"]
		" AAAA": ["2001:db8::1"]
		" MX": ["10 mx.example.org."]
		" TXT": ["\"v=spf1 include:spf.example.org -all\""]
		"www A": ["192.0.2.1"]
		"www AAAA": ["2001:db8::1"]
		"_dmarc TXT": ["\"v=DMARC1; p=reject;\""]
		"blog CNAME": ["example.net."]
	}
}

let jsonZonesFromBuilders = {
	zones: "example.org": (record.#AddressRRSets & {in: {
		subnames: ["", "www"]
		a:    "192.0.2.1"
		aaaa: "2001:db8::1"
	}}).out
	zones: "example.org": (record.#MXRRSet & {in: hosts: [{priority: 10, host: "mx.example.org."}]}).out
	zones: "example.org": (record.#TXTRRSet & {in: values: [
		(record.#TXTRRValueSPF & {in: {include: ["spf.example.org"], all: record.#SPFFail}}).out,
	]}).out
	zones: "example.org": (record.#DMARCRRSet & {in: p: record.#DMARCReject}).out
	zones: "example.org": (record.#CNAMERRSet & {in: {subname: "blog", target: "example.net."}}).out
	tokenPolicies: {}
}
