// Package docs exposes the metadata needed to serve the OpenAPI 3.1
// specification for the TrustCheck API: the location of the specification
// file and the HTTP media type it is served with.
package docs

const (
	// OpenAPIYAML is the location of the OpenAPI 3.1 specification,
	// relative to the API module root (apps/api/openapi.yaml).
	OpenAPIYAML = "openapi.yaml"

	// OpenAPIPath is the HTTP path at which the specification is served.
	OpenAPIPath = "/openapi.yaml"

	// DocsPath is the HTTP path at which the interactive Swagger UI is served.
	DocsPath = "/docs"

	// SwaggerUITitle is the page title rendered by the Swagger UI.
	SwaggerUITitle = "TrustCheck API"
)

const yamlContentType = "text/yaml; charset=utf-8"

// YAMLContentType returns the media type used to serve the OpenAPI document.
func YAMLContentType() string {
	return yamlContentType
}
