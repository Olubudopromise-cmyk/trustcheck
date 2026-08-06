// Package verifier provides input-specific verification engines.
//
// Each engine exposes a VerifyXxx function that returns a Result. The /verify
// endpoint dispatches to the matching engine based on the classifier output;
// input types without a dedicated engine keep returning the default
// "classified" mock result. This keeps the verification layer pluggable: every
// new engine (email, ip, phone, ...) plugs in the same way without touching
// the endpoint.
package verifier

// Result is the outcome of a verification engine.
type Result struct {
	Status     string
	TrustScore int
	Summary    string
}
