BINARY_NAME=miqa
BUILD_DIR=bin
INSTALL_DIR=$(HOME)/.local/bin

.PHONY: all build clean test install run help

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/miqa
	@echo "✓ Binari $(BINARY_NAME) berjaya dibina dalam $(BUILD_DIR)/"

install: build
	@mkdir -p $(INSTALL_DIR)
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "✓ miqa dipasang ke $(INSTALL_DIR)/$(BINARY_NAME)"

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR) miqa

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

help:
	@echo "Arahan Makefile:"
	@echo "  make build   - Membina binari miqa"
	@echo "  make install - Membina dan memasang ke ~/.local/bin/miqa"
	@echo "  make test    - Menjalankan unit tests"
	@echo "  make run     - Membina dan menjalankan miqa"
	@echo "  make clean   - Membersihkan fail binari"
