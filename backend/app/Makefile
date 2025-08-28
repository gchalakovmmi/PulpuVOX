# Binary name
BINARY_NAME := app
BINARY_PATH := bin/$(BINARY_NAME)

.PHONY: build run tidy clean generate

build: generate
		@echo "Building binary..."
		mkdir -p bin
		CGO_ENABLED=0 go build -o $(BINARY_PATH) ./cmd/app

run: generate
		@echo "Starting server..."
		env $$(grep -v '^#' env.vars | xargs) ./$(BINARY_PATH)

tidy:
		@echo "Organizing dependencies..."
		go mod tidy

clean:
		@echo "Cleaning binaries and generated files..."
		rm -rf bin
		find . -name "*_templ.go" -delete

generate:
		@echo "Generating Templ files..."
		templ generate

install-tools:
		@echo "Installing development tools..."
		go install github.com/a-h/templ/cmd/templ@latest
