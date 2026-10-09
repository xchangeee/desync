package record

import (
	"list"
	"regexp"
	"strings"
)

// DKIMRRSet generates a deSEC TXT record set with a DKIM public key (RFC 6376)
// from the selector and the key of a mail provider.
//
// Selector and key are taken as given by the service that signs mail for the
// zone, e.g. a mail provider; the key is copied in one piece without quotes or line
// breaks.
//
// The record set applies to the subname <selector>._domainkey. A key that is
// not base64 is an error. The version tag is optional and only written if set,
// so records copied from a provider without it stay unchanged.
//
// Usage: fill in "in", read the result from "out", and unify it with the zone,
// see core.#Zones:
//
//	zones: "example.org": (#DKIMRRSet & {in: {selector: "default", p: "MIIB..."}}).out
#DKIMRRSet: {
	X1="in": {
		// chosen by the mail provider
		selector: string
		k:        #DKIMKeyType | *#DKIMKeyRSA
		// base64 encoded public key
		p:  string
		v?: "DKIM1"
	}

	// tags of the record in the order v, k, p; v only if set
	let Tags = list.Concat([
		[if X1.v != _|_ {"v=\(X1.v)"}],
		["k=\(X1.k)", "p=\(X1.p)"],
	])

	// output: one record set with one record, or an error for a key that is not
	// base64
	if !regexp.Match("^[A-Za-z0-9+/]+=*$", X1.p) {
		out: error("DKIM key of selector \"\(X1.selector)\" is not base64")
	}
	if regexp.Match("^[A-Za-z0-9+/]+=*$", X1.p) {
		out: (#TXTRRSet & {in: {
			subname: "\(X1.selector)._domainkey"
			values: [strings.Join(Tags, "; ")]
		}}).out
	}
}

// Key type of a DKIM key, see #DKIMRRSet
#DKIMKeyType: #DKIMKeyRSA | #DKIMKeyEd25519

#DKIMKeyRSA:     "rsa"
#DKIMKeyEd25519: "ed25519"
