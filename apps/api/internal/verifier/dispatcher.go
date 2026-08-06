package verifier

import "github.com/pamierin/trustcheck/apps/api/internal/classifier"

// Verifier is the contract every verification engine implements. An engine
// receives the user input and returns a Result. Engines are registered with the
// default dispatcher by InputType (see Verify).
type Verifier interface {
	Verify(input string) Result
}

// placeholderVerifier is the engine used for every type that does not yet have
// a real implementation.
type placeholderVerifier struct{}

func (placeholderVerifier) Verify(string) Result {
	return Result{
		Status:     "not_implemented",
		TrustScore: 0,
		Summary:    "Verification engine not implemented yet.",
	}
}

// domainVerifier adapts the existing domain verification engine to the
// Verifier interface so it plugs into the dispatcher unchanged.
type domainVerifier struct{}

func (domainVerifier) Verify(input string) Result {
	return VerifyDomain(input)
}

// verifiers maps each classifier InputType to the engine that handles it.
// Adding a new engine is a single registration in this map.
var verifiers = map[classifier.InputType]Verifier{
	classifier.TypeDomain:  domainVerifier{},
	classifier.TypeURL:     urlVerifier{},
	classifier.TypeEmail:   emailVerifier{},
	classifier.TypeIPv4:    ipVerifier{},
	classifier.TypeIPv6:    ipVerifier{},
	classifier.TypePhone:   phoneVerifier{},
	classifier.TypeCompany: placeholderVerifier{},
	classifier.TypeUnknown: placeholderVerifier{},
}

// Verify dispatches the input to the verifier registered for inputType.
// Types without a registered engine fall back to the placeholder verifier.
func Verify(inputType classifier.InputType, input string) Result {
	v, ok := verifiers[inputType]
	if !ok {
		v = placeholderVerifier{}
	}
	return v.Verify(input)
}
