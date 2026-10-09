// Tests for the CUE packages in schema.
//
// CUE has no test runner, so the tests are plain CUE: each case is a field of
// "tests" or "rejects" in this package, one file per tested package, and cue
// vet evaluates all of them and fails on every case that does not hold:
//
//	cue vet -c ./test/schema/
//
// This is what "make schema" runs. The tests live outside schema, so the
// schema packages stay free of test fields for everyone importing them; they
// import the schema packages like a config would. Case names share one
// namespace, so they start with what they test, e.g. "cname" or "jsonZones".
package schema

// Tests holds cases that compare the output of the code under test with the
// expected value, by name:
//
//	tests: cname: {
//		got: (record.#CNAMERRSet & {in: {subname: "www", target: "example.net."}}).out
//		want: www: CNAME: records: ["example.net."]
//	}
//
// To see both sides of a failing case, evaluate it:
//
//	cue eval ./test/schema/ -e 'tests.cname'
tests: #Tests

#Tests: [string]: #Test

// Test compares got with want. Comparison with == is exact: extra or missing
// fields and list elements fail, field order does not matter. A mismatch fails
// vet with "conflicting values false and true" on ok.
//
// got and want are not declared, since any declaration, even got!: _, keeps an
// empty struct from a comprehension incomplete, e.g. #AddressRRSets without
// subnames (CUE v0.17). The definition is open instead and refers to them
// through the alias T; a case without got or want still fails vet with
// "undefined field".
#Test: T={
	ok: true & (T.got == T.want)
	...
}

// Rejects holds cases whose value has to be an error, by name, each written as
// a comparison with bottom:
//
//	rejects: cnameWithoutDot: (record.#CNAMERRSet & {in: {subname: "www", target: "example.net"}}).out == _|_
//
// The comparison has to happen at the case itself: a field holding the error
// would fail vet on its own. Only that there is an error is checked, not its
// message. A missing required input is incomplete, not an error, so it cannot
// be tested this way; cue vet -c fails on it instead.
rejects: #Rejects

#Rejects: [string]: true
