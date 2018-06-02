build:
	mkdir -p dist/
	go build -o dist/ooni-collector .

.PHONY: build

release:
	goreleaser --rm-dist

.PHONY: release
