BINARY_NAME=ytracker

build:
	GOARCH=amd64 GOOS=darwin go build -o bin/${BINARY_NAME}-darwin main.go
	GOARCH=amd64 GOOS=linux go build -o bin/${BINARY_NAME}-linux main.go
	GOARCH=amd64 GOOS=windows go build -o bin/${BINARY_NAME}-windows main.go

run:
	go run main.go

clean:
	go clean
	rm bin/${BINARY_NAME}-darwin
	rm bin/${BINARY_NAME}-linux
	rm bin/${BINARY_NAME}-windows

test:
	go test ./...

test_coverage:
	go test ./... -coverprofile=coverage.out

dep:
	go mod download

vet:
	go vet

lint:
	golangci-lint run --enable-all
