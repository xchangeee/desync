package record

// AddressRRSets generates deSEC A and AAAA record sets that point the given
// subnames to the IPv4 and IPv6 address of a server.
//
// Usage: fill in "in", read the result from "out", and unify it with the zone,
// see core.#Zones:
//
//	zones: "example.org": (#AddressRRSets & {in: {
//		subnames: ["", "www"]
//		a:    "192.0.2.1"
//		aaaa: "2001:db8::1"
//	}}).out
#AddressRRSets: {
	X1="in": {
		// subnames the records are for, "" for the zone apex
		subnames: [...string]
		// IPv4 and IPv6 address of the server
		a:    string
		aaaa: string
	}

	// output: two record sets per subname
	out: {
		for name in X1.subnames {
			(name): {
				A: records: [X1.a]
				AAAA: records: [X1.aaaa]
			}
		}
	}
}
