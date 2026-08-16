.PHONY: k8s docker-build openapi-local
BINARY_NAME=ytracker
DOCKER_IMAGE_NAME=youtube-tracker

generate:
	go generate

# Local copy of the spec with servers replaced by the HTTP Client {{url}} variable,
# so requests generated from it in the IDE target the environment picked at run time
openapi-local:
	yq '.servers = [{"url": "http://{{host}}"}]' openapi.yaml > openapi.local.yaml

build: generate compile-all
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux ./cmd/app/main.go

full_build: build vet lint database_test integration_test
	GOARCH=amd64 GOOS=darwin go build -o bin/${BINARY_NAME}-darwin ./cmd/app/main.go
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux ./cmd/app/main.go
	GOARCH=amd64 GOOS=windows go build -o bin/${BINARY_NAME}-windows ./cmd/app/main.go

full_build_with_thirdparty: build vet lint database_test integration_test thirdparty_test
	GOARCH=amd64 GOOS=darwin go build -o bin/${BINARY_NAME}-darwin ./cmd/app/main.go
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux ./cmd/app/main.go
	GOARCH=amd64 GOOS=windows go build -o bin/${BINARY_NAME}-windows ./cmd/app/main.go

run:
	go run main.go

clean:
	go clean
	rm -f bin/*
	rm -f internal/api/*

compile-all: generate
	go build ./cmd/... ./internal/...
	go test -tags=integration,database,thirdparty -count=0 ./cmd/... ./internal/...

test:
	go test -v ./...

extract-schema:
	go run AtlasSqlExtractor.go

integration_test: extract-schema
	go test -v -tags=integration,database ./...

database_test: extract-schema
	go test -v -tags=database ./...

thirdparty_test:
	go test -v -tags=thirdparty ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

dep:
	go mod download

vet:
	go vet

lint:
	golangci-lint run

docker-build:
	docker build -t ${DOCKER_IMAGE_NAME} .

k8s:
	./apply-k8s.sh
