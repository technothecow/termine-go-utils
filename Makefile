PHONY := vendor lint test fix

lint:
	golangci-lint run -c .golangci.yml -v ./...

fix:
	go mod tidy
	golangci-lint run -c .golangci.yml -v ./... --fix

test:
	go test ./...
