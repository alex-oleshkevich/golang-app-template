package api

import _ "embed"

// RawSpec is openapi.yaml as written. GetSpecJSON is not used for publishing:
// the generator rewrites operationIds.
//
//go:embed openapi.yaml
var RawSpec []byte
