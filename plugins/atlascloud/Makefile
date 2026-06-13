
files := $(wildcard *.go)

all: $(files)
	go build -o atlascloud.out $(files)

prod: $(files)
	go build -o atlascloud.out -ldflags "-s -w" $(files)

release: prod
	goreleaser build -f .goreleaser.yaml --clean --snapshot

clean:
	rm -f atlascloud.out

.PHONY: all clean
