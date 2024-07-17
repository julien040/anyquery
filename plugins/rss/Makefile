
files := $(wildcard *.go)

all: $(files)
	go build -o rss.out $(files)

prod: $(files)
	go build -o rss.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
