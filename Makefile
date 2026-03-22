.PHONY: k8s docker-build argocd-install argocd-password
BINARY_NAME=ytracker
DOCKER_IMAGE_NAME=youtube-tracker

generate:
	go generate

build: generate test vet lint
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux ./cmd/app/main.go

full_build: build integration_test
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

argocd-install:
	./install-argocd.sh

argocd-password:
	@kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d && echo