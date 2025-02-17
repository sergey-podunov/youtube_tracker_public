BINARY_NAME=ytracker

generate:
	go generate

build: generate test
	GOARCH=amd64 GOOS=darwin go build -o cmd/${BINARY_NAME}-darwin main.go
	GOARCH=amd64 GOOS=linux go build -o cmd/${BINARY_NAME}-linux main.go
	GOARCH=amd64 GOOS=windows go build -o cmd/${BINARY_NAME}-windows main.go

run:
	go run main.go

clean:
	go clean
	rm -f bin/*
	rm -f internal/api/*

test:
	go test -tags=integration ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

dep:
	go mod download

vet:
	go vet

lint:
	golangci-lint run --enable-all
