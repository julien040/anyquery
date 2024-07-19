
files := $(wildcard *.go)

all: $(files)
	go build -o notes.out $(files)

prod: $(files)
	go build -o notes.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
