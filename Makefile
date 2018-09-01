release:
	@echo "** Releasing version $(VERSION)..."
	@echo "** Building..."
	@$(MAKE) build
	@echo "** Tagging and pushing..."
	@git tag -a $(VERSION) -m "$(VERSION)"
	@git push --tags
.PHONY: release

build:
	for GOOS in darwin linux openbsd windows; do
		for GOARCH in 386 amd64; do
			GOOS=$1
			GOARCH=$2
			filename="dnote-${GOOS}-${GOARCH}"

			echo "Building $filename"
			GOOS=$GOOS GOARCH=$GOARCH go build -o "${filename}" -ldflags "-X main.apiEndpoint=https://api.dnote.io"
		done
	done
.PHONY: build

clean:
	@git clean -f
.PHONY: clean
