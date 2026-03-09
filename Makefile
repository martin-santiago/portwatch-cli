BINARY = portwatch
BUILD_DIR = build

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
	@cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed to /usr/local/bin/$(BINARY)"

uninstall:
	@rm -f /usr/local/bin/$(BINARY)
	@echo "Uninstalled $(BINARY)."
