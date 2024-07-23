
files := $(wildcard *.go)

all: $(files)
	go build -o airtable.out $(files)

prod: $(files)
	go build -o airtable.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
