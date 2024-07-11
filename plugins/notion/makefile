files := $(wildcard *.go)

all: $(files)
	go build -o notion.out $(files)

prod: $(files)
	go build -o notion.out -ldflags "-s -w" $(files)

release: $(files)
	goreleaser release --clean --snapshot
	../store-manager/store-manager.out -p notion -c manifest.toml

clean:
	rm -f notion.out

.PHONY: all clean