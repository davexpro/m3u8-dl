.PHONY: build
build:
	mkdir -p output
    CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -extldflags '-static'" -o output/m3u8-dl_darwin_amd64
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -extldflags '-static'" -o output/m3u8-dl_linux_amd64
    CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -extldflags '-static'" -o output/m3u8-dl_darwin_arm
