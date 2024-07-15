
files := $(wildcard *.go)

all: $(files)
	go build -o pocket.out $(files)

prod: $(files)
	go build -o pocket.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
