
files := $(wildcard *.go)

all: $(files)
	go build -o asana.out $(files)

prod: $(files)
	go build -o asana.out -ldflags "-s -w" $(files)

release: prod
	goreleaser build -f .goreleaser.yaml --clean --snapshot

clean:
	rm -f asana.out

.PHONY: all clean
