
files := $(wildcard *.go)

all: $(files)
	go build -o salesforce_custom.out $(files)

prod: $(files)
	go build -o salesforce_custom.out -ldflags "-s -w" $(files)

clean:
	rm -f salesforce_custom.out

.PHONY: all clean
