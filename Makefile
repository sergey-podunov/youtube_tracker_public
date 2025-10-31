.PHONY: k8s docker-build
BINARY_NAME=ytracker
DOCKER_IMAGE_NAME=youtube-tracker

generate:
	go generate

build: generate integration_test vet lint
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
	go test -v ./...

integration_test:
	go test -v -tags=integration,database ./...

database_test:
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

docker-build: integration_test vet lint
	docker build -t ${DOCKER_IMAGE_NAME} .

atlas-schema:
	go run -tags=atlas_schema AtlasSchemaGenerator.go

k8s: atlas-schema
	./apply-k8s.sh