package tool

import (
	"encoding/json"

	"github.com/xchangeee/desync/schema/core@v0"
)

// DesyncJsonFromZones renders #Desec as the JSON config of desync.
//
// desync expects each zone as a list of record sets with subname and type; they
// are taken from the path in #Zones.
//
// Usage: fill in "in", read the result from "out":
//
//	(#DesyncJsonFromZones & {in: desec}).out
#DesyncJsonFromZones: {
	X1="in": core.#Desec

	let Desync = core.#Desync & {
		domains: [
			for Zone, subnames in X1.zones {
				name: Zone
				rrsets: [
					for Subname, types in subnames
					for Type, rrset in types {
						subname: Subname
						type:    Type
						records: rrset.records
					},
				]
			},
		]
		tokenPolicies: [
			for Id, Policies in X1.tokenPolicies {
				tokenId:  Id
				policies: Policies
			},
		]
	}

	// output: the config as JSON
	out: json.Marshal(Desync)
}
