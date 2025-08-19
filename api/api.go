package api

//go:generate go run -modfile=../tools/go.mod github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=config.yaml api.yaml

func Float(v float64) *float64 {
	return &v
}
