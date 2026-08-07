// Package spec embeds the OpenAPI 3.1 specification into the compiled binary
// so both the local development server and the Netlify Function can serve it
// without reading a file from disk at runtime.
package spec

import _ "embed"

//go:embed openapi.yaml
var YAML []byte
