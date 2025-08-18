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
	# env $$(cat vars.env | xargs) go run main.go
	# env $$(cat vars.env | xargs) ./app
	env $$(grep -v '^#' vars.env | xargs) ./$(BINARY)
