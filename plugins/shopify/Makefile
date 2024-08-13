
files := $(wildcard *.go)

all: $(files)
	go build -o shopify.out $(files)

prod: $(files)
	go build -o shopify.out -ldflags "-s -w" $(files)

clean:
	rm -f shopify.out

.PHONY: all clean
