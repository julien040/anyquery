
files := $(wildcard *.go)

all: $(files)
	go build -o git.out $(files)

prod: $(files)
	go build -o git.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
