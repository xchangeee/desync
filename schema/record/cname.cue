package record

import (
	"strings"
)

// CNAMERRSet generates a deSEC CNAME record set that points the given subname
// to another host.
//
// A target without a trailing dot is an error, since deSEC expects fully
// qualified names.
//
// Usage: fill in "in", read the result from "out", and unify it with the zone,
// see core.#Zones:
//
//	zones: "example.org": (#CNAMERRSet & {in: {subname: "www", target: "example.net."}}).out
#CNAMERRSet: {
	X1="in": {
		subname: string
		// fully qualified, with a trailing dot
		target: string
	}

	// output: one record set with one record, or an error for a target without
	// a trailing dot
	if !strings.HasSuffix(X1.target, ".") {
		out: error("CNAME target \"\(X1.target)\" must end with a dot")
	}
	if strings.HasSuffix(X1.target, ".") {
		out: (X1.subname): CNAME: records: [X1.target]
	}
}
