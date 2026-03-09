BINARY = pw
BUILD_DIR = build

# Use ~/.local/bin on Linux (no sudo), /usr/local/bin on macOS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	INSTALL_DIR = /usr/local/bin
else
	INSTALL_DIR = $(HOME)/.local/bin
endif

.PHONY: build run clean install uninstall

build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY) .
	@echo "Built $(BUILD_DIR)/$(BINARY)"

run: build
	@$(BUILD_DIR)/$(BINARY)

clean:
	@rm -rf $(BUILD_DIR)
	@echo "Cleaned."

install: build
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"
	@echo "Run 'pw' from any directory."

uninstall:
	@rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Uninstalled $(BINARY) from $(INSTALL_DIR)."
