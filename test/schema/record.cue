// Tests for the record set builders of schema/record: "tests" holds the record
// sets each builder generates, "rejects" the inputs it has to turn into an
// error, e.g. a host without a trailing dot.
//
// A missing required input such as the DMARC policy is incomplete, not an
// error, so it is not covered here.
package schema

import (
	"strings"

	"github.com/xchangeee/desync/schema/record@v0"
)

// address

tests: address: {
	got: (record.#AddressRRSets & {in: {
		subnames: ["", "www"]
		a:    "192.0.2.1"
		aaaa: "2001:db8::1"
	}}).out
	want: {
		"": {A: records: ["192.0.2.1"], AAAA: records: ["2001:db8::1"]}
		www: {A: records: ["192.0.2.1"], AAAA: records: ["2001:db8::1"]}
	}
}

tests: addressNoSubnames: {
	got: (record.#AddressRRSets & {in: {subnames: [], a: "192.0.2.1", aaaa: "2001:db8::1"}}).out
	want: {}
}

// CNAME

tests: cname: {
	got: (record.#CNAMERRSet & {in: {subname: "www", target: "example.net."}}).out
	want: www: CNAME: records: ["example.net."]
}

rejects: cnameWithoutDot: (record.#CNAMERRSet & {in: {subname: "www", target: "example.net"}}).out == _|_

// TXT

tests: txtValue: {
	got: (record.#TXTRRValue & {in: "token"}).out
	want: "\"token\""
}

tests: txtValueEmpty: {
	got: (record.#TXTRRValue & {in: ""}).out
	want: "\"\""
}

// exactly one string at the limit
tests: txtValue255: {
	got: (record.#TXTRRValue & {in: strings.Repeat("a", 255)}).out
	want: "\"\(strings.Repeat("a", 255))\""
}

// split into strings of at most 255 characters
tests: txtValue600: {
	got: (record.#TXTRRValue & {in: strings.Repeat("a", 255) + strings.Repeat("b", 255) + strings.Repeat("c", 90)}).out
	want: "\"\(strings.Repeat("a", 255))\" \"\(strings.Repeat("b", 255))\" \"\(strings.Repeat("c", 90))\""
}

rejects: txtValueQuote: (record.#TXTRRValue & {in: "a\"b"}).out == _|_
rejects: txtValueBackslash: (record.#TXTRRValue & {in: "a\\b"}).out == _|_

tests: txt: {
	got: (record.#TXTRRSet & {in: values: ["one", "two"]}).out
	want: "": TXT: records: ["\"one\"", "\"two\""]
}

tests: txtSubname: {
	got: (record.#TXTRRSet & {in: {subname: "_example", values: ["token"]}}).out
	want: "_example": TXT: records: ["\"token\""]
}

rejects: txtQuote: (record.#TXTRRSet & {in: values: ["a\"b"]}).out == _|_

// SPF

tests: spf: {
	got: (record.#TXTRRValueSPF & {in: {
		include: ["spf.example.org"]
		ip4: ["192.0.2.0/24"]
		ip6: ["2001:db8::/32"]
		all: record.#SPFFail
	}}).out
	want: "v=spf1 include:spf.example.org ip4:192.0.2.0/24 ip6:2001:db8::/32 -all"
}

tests: spfSoftfail: {
	got: (record.#TXTRRValueSPF & {in: all: record.#SPFSoftfail}).out
	want: "v=spf1 ~all"
}

tests: spfNeutral: {
	got: (record.#TXTRRValueSPF & {in: all: record.#SPFNeutral}).out
	want: "v=spf1 ?all"
}

rejects: spfPass: (record.#TXTRRValueSPF & {in: all: "pass"}).out == _|_

// the documented use together with other TXT values of the zone apex
tests: spfInTXT: {
	got: (record.#TXTRRSet & {in: values: [
		"google-site-verification=token",
		(record.#TXTRRValueSPF & {in: {include: ["spf.example.org"], all: record.#SPFSoftfail}}).out,
	]}).out
	want: "": TXT: records: [
		"\"google-site-verification=token\"",
		"\"v=spf1 include:spf.example.org ~all\"",
	]
}

// DMARC

tests: dmarc: {
	got: (record.#DMARCRRSet & {in: p: record.#DMARCReject}).out
	want: "_dmarc": TXT: records: ["\"v=DMARC1; p=reject;\""]
}

tests: dmarcRua: {
	got: (record.#DMARCRRSet & {in: {p: record.#DMARCQuarantine, rua: ["mailto:a@example.org", "mailto:b@example.org"]}}).out
	want: "_dmarc": TXT: records: ["\"v=DMARC1; p=quarantine; rua=mailto:a@example.org,mailto:b@example.org;\""]
}

rejects: dmarcUnknownPolicy: (record.#DMARCRRSet & {in: p: "drop"}).out == _|_

// DKIM

tests: dkim: {
	got: (record.#DKIMRRSet & {in: {selector: "default", p: "MIIBIjANBgkq+/A="}}).out
	want: "default._domainkey": TXT: records: ["\"k=rsa; p=MIIBIjANBgkq+/A=\""]
}

tests: dkimVersion: {
	got: (record.#DKIMRRSet & {in: {selector: "mail", k: record.#DKIMKeyEd25519, p: "MCowBQ", v: "DKIM1"}}).out
	want: "mail._domainkey": TXT: records: ["\"v=DKIM1; k=ed25519; p=MCowBQ\""]
}

rejects: dkimNotBase64: (record.#DKIMRRSet & {in: {selector: "mail", p: "not base64!"}}).out == _|_
rejects: dkimUnknownType: (record.#DKIMRRSet & {in: {selector: "mail", k: "dsa", p: "MCowBQ"}}).out == _|_

// MX

tests: mx: {
	got: (record.#MXRRSet & {in: hosts: [
		{priority: 10, host: "mx01.example.org."},
		{priority: 20, host: "mx02.example.org."},
	]}).out
	want: "": MX: records: ["10 mx01.example.org.", "20 mx02.example.org."]
}

tests: mxSubname: {
	got: (record.#MXRRSet & {in: {subname: "mail", hosts: [{priority: 0, host: "mx.example.org."}]}}).out
	want: mail: MX: records: ["0 mx.example.org."]
}

rejects: mxWithoutDot: (record.#MXRRSet & {in: hosts: [{priority: 10, host: "mx.example.org"}]}).out == _|_

// CAA

tests: caa: {
	got: (record.#CAARRSet & {in: {
		issue: ["letsencrypt.org"]
		issuewild: [";"]
		iodef: ["mailto:security@example.org"]
	}}).out
	want: "": CAA: records: [
		"0 issue \"letsencrypt.org\"",
		"0 issuewild \";\"",
		"0 iodef \"mailto:security@example.org\"",
	]
}

tests: caaSubname: {
	got: (record.#CAARRSet & {in: {subname: "www", issue: ["letsencrypt.org"]}}).out
	want: www: CAA: records: ["0 issue \"letsencrypt.org\""]
}

rejects: caaQuote: (record.#CAARRSet & {in: issue: ["a\"b"]}).out == _|_
