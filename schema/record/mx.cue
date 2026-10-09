package record

import (
	"strings"
)

// MXRRSet generates a deSEC MX record set from a list of mail servers. This
// writes one record "<priority> <host>" per server.
//
// The record set applies to the given subname, the zone apex by default. A host
// without a trailing dot is an error, since deSEC expects fully qualified names.
//
// Usage: fill in "in", read the result from "out", and unify it with the zone,
// see core.#Zones:
//
//	zones: "example.org": (#MXRRSet & {in: hosts: [{priority: 10, host: "mx01.example.org."}]}).out
#MXRRSet: {
	X1="in": {
		// subname the records are for, the zone apex by default
		subname: string | *""
		hosts: [...{
			// lower values are tried first
			priority: int
			// fully qualified, with a trailing dot
			host: string
		}]
	}

	// output: one record set with one record per host
	out: (X1.subname): MX: records: [
		for h in X1.hosts {
			if !strings.HasSuffix(h.host, ".") {
				error("MX host \"\(h.host)\" must end with a dot")
			}
			if strings.HasSuffix(h.host, ".") {"\(h.priority) \(h.host)"}
		},
	]
}
