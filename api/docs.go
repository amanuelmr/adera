package apidocs

import _ "embed"

// OpenAPIYAML is the embedded API contract served by the application.
//
//go:embed openapi.yaml
var OpenAPIYAML []byte
