.PHONY: k8s
BINARY_NAME=ytracker

generate:
	go generate

build: generate test
	GOARCH=amd64 GOOS=darwin go build -o bin/${BINARY_NAME}-darwin ./cmd/app/main.go
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux ./cmd/app/main.go
	GOARCH=amd64 GOOS=windows go build -o bin/${BINARY_NAME}-windows ./cmd/app/main.go

run:
	go run main.go

clean:
	go clean
	rm -f bin/*
	rm -f internal/api/*

test:
	go test -v -tags=database ./...

integration_test:
	go test -v -tags=integration ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

dep:
	go mod download

vet:
	go vet

lint:
	golangci-lint run --enable-all

atlas-schema:
	go run -tags=atlas_schema AtlasSchemaGenerator.go

k8s: atlas-schema
	./apply-k8s.sh
