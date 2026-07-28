VERSION ?= dev
DIST ?= dist
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: test vet build release clean

test:
	go test ./...

vet:
	go vet ./...

build:
	@mkdir -p $(DIST)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/ciyuanmax ./cmd/ciyuanmax

release:
	@mkdir -p $(DIST)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/ciyuanmax-darwin-arm64 ./cmd/ciyuanmax
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/ciyuanmax-darwin-amd64 ./cmd/ciyuanmax
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/ciyuanmax-windows-amd64.exe ./cmd/ciyuanmax
	@shasum -a 256 $(DIST)/ciyuanmax-* > $(DIST)/SHA256SUMS.txt

clean:
	rm -rf $(DIST)
