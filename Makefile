.PHONY: generate
generate:
	@go generate ./...

.PHONY: run
## Runs the application on the local machine
run: generate
	@go run cmd/main/main.go

.PHONY: format
## Run go fmt against the codebase
format:
	@echo "formatting..."
	@gofumpt -l -w .
