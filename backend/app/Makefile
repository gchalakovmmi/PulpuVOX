BINARY := app
.PHONY: clear compile run

clear:
	@clear

compile:
	@echo "Generating templ files ..."
	go tool templ generate ./...
	@echo "Building binary ..."
	CGO_ENABLED=0 go build -o $(BINARY) .
	@echo "Cleaning _templ.go files ..."
	find pages/ -name "*_templ.go" -exec rm {} \;

run:
	@echo "Runing $(BINARY)"
	BACKEND_PORT=8080 ./app
