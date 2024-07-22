
files := $(wildcard *.go)

all: $(files)
	go build -o google_sheets.out $(files)

prod: $(files)
	go build -o google_sheets.out -ldflags "-s -w" $(files)

clean:
	rm -f google_sheets.out

.PHONY: all clean
