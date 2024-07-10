files := $(wildcard *.go)

all: $(files)
	go build -o spotify.out $(files)

prod: $(files)
	go build -o spotify.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean