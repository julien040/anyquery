
files := $(wildcard *.go)

all: $(files)
	go build -o hacker_news.out $(files)

prod: $(files)
	go build -o hacker_news.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
