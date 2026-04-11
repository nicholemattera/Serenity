BINARY_NAME=serenity
BUILD_DIR=bin

.PHONY: build run pretty lint clean

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

run:
	go run main.go

test:
	go test -v ./...

pretty:
	gofmt -w .

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
