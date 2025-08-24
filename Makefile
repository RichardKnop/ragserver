.PHONY: generate
generate:
	@go generate ./...

.PHONY: format
## Run go fmt against the codebase
format:
	@echo "formatting..."
	@gofumpt -l -w .
