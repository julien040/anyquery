
files := $(wildcard *.go)

all: $(files)
	go build -o readwise.out $(files)

prod: $(files)
	go build -o readwise.out -ldflags "-s -w" $(files)

release: prod
	goreleaser build -f .goreleaser.yaml --clean --snapshot

clean:
	rm -f readwise.out

.PHONY: all clean
