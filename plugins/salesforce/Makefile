
files := $(wildcard *.go)

all: $(files)
	go build -o salesforce.out $(files)

prod: $(files)
	go build -o salesforce.out -ldflags "-s -w" $(files)

clean:
	rm -f salesforce.out

.PHONY: all clean
