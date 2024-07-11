
files := $(wildcard *.go)

all: $(files)
	go build -o homebrew.out $(files)

prod: $(files)
	go build -o homebrew.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
