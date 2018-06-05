build:
	mkdir -p dist/
	go build -o dist/ooni-collector .

.PHONY: build

release:
	GITHUB_TOKEN=`cat .GITHUB_TOKEN` goreleaser --rm-dist

.PHONY: release
