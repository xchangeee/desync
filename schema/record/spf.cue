package record

import (
	"list"
	"strings"
)

// TXTRRValueSPF generates an SPF policy (RFC 7208) as the value of a TXT
// record from the allowed senders and the result for all other senders.
//
// Senders are given as include domains, ip4 and ip6 addresses; they are written
// in this order. The result for all other senders is one of #SPFAll and has to
// be given; "pass" is not offered.
//
// Usage: fill in "in", read the result from "out", and pass it to #TXTRRSet
// together with the other TXT values of the subname:
//
//	(#TXTRRSet & {in: values: [
//		"google-site-verification=...",
//		(#TXTRRValueSPF & {in: {include: ["spf.example.org"], all: #SPFSoftfail}}).out,
//	]}).out
#TXTRRValueSPF: {
	X1="in": {
		// domains whose SPF policy is included, e.g. a mail provider
		include: [...string]
		// addresses or networks allowed to send directly
		ip4: [...string]
		ip6: [...string]
		// result for all other senders
		all!: #SPFAll
	}

	// qualifier written in front of "all" for each result
	let Qualifier = {(#SPFFail): "-", (#SPFSoftfail): "~", (#SPFNeutral): "?"}

	// output: the policy as one string
	out: strings.Join(list.Concat([
		["v=spf1"],
		[for d in X1.include {"include:\(d)"}],
		[for a in X1.ip4 {"ip4:\(a)"}],
		[for a in X1.ip6 {"ip6:\(a)"}],
		["\(Qualifier[X1.all])all"],
	]), " ")
}

// Result of the SPF check for all senders not listed, see #TXTRRValueSPF
#SPFAll: #SPFFail | #SPFSoftfail | #SPFNeutral

// rejected
#SPFFail: "fail"

// accepted, but marked as suspicious
#SPFSoftfail: "softfail"

// no statement about the sender
#SPFNeutral: "neutral"
