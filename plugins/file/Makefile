
files := $(wildcard *.go)

all: $(files)
	go build -o file.out $(files)

prod: $(files)
	go build -o file.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
