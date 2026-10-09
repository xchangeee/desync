package record

import (
	"strings"
)

// DMARCRRSet generates a deSEC TXT record set with the DMARC policy (RFC 7489)
// of the zone.
//
// The policy has to be given, there is no default. Aggregate reports are only
// requested if an address is given.
//
// The record set applies to the subname _dmarc.
//
// Usage: fill in "in", read the result from "out", and unify it with the zone,
// see core.#Zones:
//
//	zones: "example.org": (#DMARCRRSet & {in: p: #DMARCReject}).out
#DMARCRRSet: {
	X1="in": {
		// What receivers do with mail that fails DMARC
		p!: #DMARCPolicy
		// mailto: URIs for aggregate reports
		rua: [...string]
	}

	// A missing policy cannot get its own error() message: the definition itself
	// has no policy, so the message would fail every use of #DMARCRRSet. CUE
	// reports it as "in: invalid interpolation" on the "values" line below.
	let Rua = strings.Join([if len(X1.rua) > 0 {" rua=\(strings.Join(X1.rua, ","));"}], "")

	// output: one record set with one record
	out: (#TXTRRSet & {in: {
		subname: "_dmarc"
		values: ["v=DMARC1; p=\(X1.p);\(Rua)"]
	}}).out
}

// What receivers do with mail that fails DMARC, see #DMARCRRSet
#DMARCPolicy: #DMARCReject | #DMARCQuarantine | #DMARCNone

// rejected
#DMARCReject: "reject"

// delivered as spam
#DMARCQuarantine: "quarantine"

// delivered, only reported
#DMARCNone: "none"
