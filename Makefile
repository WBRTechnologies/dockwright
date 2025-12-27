BINARY_NAME=dockwright
INSTALL_PATH=/usr/local/bin
CHARTS_PATH=/usr/local/share/dockwright/charts

.PHONY: all build install clean deps package

all: build

deps:
	@echo "📦 Downloading dependencies..."
	@go mod tidy
	@go mod download
	@echo "✅ Dependencies ready"
	@echo "───────────────────────────────────────────────────────────────────────"

build: deps
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p build
	@go build -o build/$(BINARY_NAME) main.go
	@echo "✅ Build complete: build/$(BINARY_NAME)"
	@echo "───────────────────────────────────────────────────────────────────────"

install: build package
	@echo "📦 Installing $(BINARY_NAME)..."
	@sudo mv build/$(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Installed $(BINARY_NAME) to $(INSTALL_PATH)"
	@echo "───────────────────────────────────────────────────────────────────────"
	@$(MAKE) clean

package:
	@echo "📦 Packaging Helm charts..."
	@sudo cp -r ./base-helm-charts/stateful/* $(CHARTS_PATH)/stateful/
	@sudo cp -r ./base-helm-charts/stateless/* $(CHARTS_PATH)/stateless/
	@echo "✅ Charts packaged and moved to $(CHARTS_PATH)"
	@echo "───────────────────────────────────────────────────────────────────────"

clean:
	@echo "🧹 Cleaning up..."
	@rm -rf build/
	@echo "✅ Clean complete"
	@echo "───────────────────────────────────────────────────────────────────────"
