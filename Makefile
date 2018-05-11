build:
	mkdir -p dist/
	go build -o dist/ooni-collector .

.PHONY: build
