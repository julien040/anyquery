
files := $(wildcard *.go)

all: $(files)
	go build -o carddav.out $(files)

prod: $(files)
	go build -o carddav.out -ldflags "-s -w" $(files)

release: prod
	goreleaser build -f .goreleaser.yaml --clean --snapshot

test:
	go test -v ./...

integration-test:
	./test.sh

clean:
	rm -f carddav.out

.PHONY: all clean test integration-test
