package record

import (
	"list"
	"strings"
)

// CAARRSet generates a deSEC CAA record set (RFC 8659) from the certificate
// authorities allowed to issue certificates for the zone.
//
// issue lists the CAs for normal certificates, issuewild those for wildcard
// certificates, iodef the URLs for reports; each entry becomes one record.
// Values with quotes are an error.
//
// The record set applies to the given subname, the zone apex by default.
//
// Usage: fill in "in", read the result from "out", and unify it with the zone,
// see core.#Zones:
//
//	zones: "example.org": (#CAARRSet & {in: issue: ["letsencrypt.org"]}).out
#CAARRSet: {
	X1="in": {
		// subname the records are for, the zone apex by default
		subname: string | *""
		// CA domains, e.g. "letsencrypt.org"
		issue: [...string]
		issuewild: [...string]
		// mailto: or https: URLs
		iodef: [...string]
	}

	let Entries = list.Concat([
		[for v in X1.issue {tag: "issue", value: v}],
		[for v in X1.issuewild {tag: "issuewild", value: v}],
		[for v in X1.iodef {tag: "iodef", value: v}],
	])

	// output: one record set with one record per entry
	out: (X1.subname): CAA: records: [
		for e in Entries {
			if strings.Contains(e.value, "\"") {
				error("CAA \(e.tag) value \"\(e.value)\" must not contain quotes")
			}
			if !strings.Contains(e.value, "\"") {"0 \(e.tag) \"\(e.value)\""}
		},
	]
}
