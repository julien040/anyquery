files := $(wildcard *.go)

all: $(files)
	go build -o notion.out $(files)

prod: $(files)
	go build -o notion.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean