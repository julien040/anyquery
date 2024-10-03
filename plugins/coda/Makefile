
files := $(wildcard *.go)

all: $(files)
	go build -o coda.out $(files)

prod: $(files)
	go build -o coda.out -ldflags "-s -w" $(files)

clean:
	rm -f coda.out

.PHONY: all clean
