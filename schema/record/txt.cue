package record

import (
	"list"
	"strings"
)

// TXTRRSet generates a deSEC TXT record set from a list of plain values. This
// quotes each value and splits it into strings of at most 255 characters, see
// #TXTRRValue.
//
// Use it for TXT records that have no more specific builder, such as domain
// verification tokens.
//
// A subname has only one TXT record set, so pass all of its values in one
// call. Two calls for the same subname fail to unify, unless their values are
// identical. At the zone apex, for example, one call holds both the
// verification tokens and the SPF policy.
//
// The record set applies to the given subname, the zone apex by default.
//
// Usage: fill in "in", read the result from "out", and unify it with the zone,
// see core.#Zones:
//
//	zones: "example.org": (#TXTRRSet & {in: {subname: "_example", values: ["token"]}}).out
#TXTRRSet: {
	X1="in": {
		// subname the records are for, the zone apex by default
		subname: string | *""
		// unquoted values, one per record
		values: [...string]
	}

	// output: one record set with one record per value
	out: (X1.subname): TXT: records: [for v in X1.values {(#TXTRRValue & {in: v}).out}]
}

// TXTRRValue generates the value of a single TXT record, as deSEC expects it in
// the records list of a record set, from a plain string.
//
// deSEC expects each TXT record as one or more quoted strings, and a single
// string holds at most 255 characters. A longer value such as a DKIM key has to
// be split into several strings, which the resolver joins back together.
// TXTRRValue does the quoting and splitting, so zone files contain the plain
// value instead of a hand-quoted and hand-split one.
//
// Quotes and backslashes would need escaping, which is not supported, so they
// are rejected instead.
//
// Usage: fill in "in", read the result from "out":
//
//	(#TXTRRValue & {in: "token"}).out
#TXTRRValue: {
	X1="in": string

	let N = len(X1)

	// output: the quoted value, or an error for quotes and backslashes
	// (error() instead of =~ on "in", which CUE drops when only "out" is read)
	if strings.ContainsAny(X1, "\"\\") {
		out: error("TXT value \"\(X1)\" must not contain quotes or backslashes")
	}
	if !strings.ContainsAny(X1, "\"\\") && N == 0 {
		out: "\"\""
	}
	if !strings.ContainsAny(X1, "\"\\") && N > 0 {
		out: strings.Join([
			for i in list.Range(0, N, 255) {
				"\"\(strings.SliceRunes(X1, i, list.Min([i + 255, N])))\""
			},
		], " ")
	}
}
