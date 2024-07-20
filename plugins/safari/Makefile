
files := $(wildcard *.go)

all: $(files)
	go build -o safari.out $(files)

prod: $(files)
	go build -o safari.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
