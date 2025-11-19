.PHONY: build clean install deps

BINARY_NAME=uptime-kuma-backup
INSTALL_PATH=/usr/local/bin

build:
	go build -o $(BINARY_NAME) .

clean:
	go clean
	rm -f $(BINARY_NAME)

install: build
	sudo mv $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)

deps:
	go mod download
	go mod tidy

run: build
	./$(BINARY_NAME)
