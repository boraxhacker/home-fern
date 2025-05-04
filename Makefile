.PHONY: clean run build

BINARY_NAME=home-fern

build: clean
	mkdir release
	GOOS=linux GOARCH=amd64 go build -o release/${BINARY_NAME}-amd64 ./cmd/${BINARY_NAME}
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o release/${BINARY_NAME}-alpine ./cmd/${BINARY_NAME}

run:
	go run ./cmd/home-fern

clean:
	go clean
	rm -rf release
