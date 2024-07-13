
files := $(wildcard *.go)

all: $(files)
	go build -o github.out $(files)

prod: $(files)
	go build -o github.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
