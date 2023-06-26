package packageB

//go:generate go run github.com/truvity/oapi-codegen/cmd/oapi-codegen -generate types,skip-prune,spec --package=packageB -o externalref.gen.go spec.yaml
