
files := $(wildcard *.go)

all: $(files)
	go build -o raindrop.out $(files)

prod: $(files)
	go build -o raindrop.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
