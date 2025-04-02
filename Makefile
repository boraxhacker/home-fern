.PHONY: clean run build

BINARY_NAME=home-fern

build: clean 
	GOOS=linux GOARCH=amd64 go build -o ${BINARY_NAME} ./cmd/home-fern

run:
	go run ./cmd/home-fern

clean:
	go clean
	rm -rf ${BINARY_NAME}
