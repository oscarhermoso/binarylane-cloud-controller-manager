package main

//go:generate ./scripts/fetch-openapi.sh
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=internal/binarylane/client.cfg.yml ./openapi.json
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=internal/binarylane/types.cfg.yml ./openapi.json
