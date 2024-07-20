
files := $(wildcard *.go)

all: $(files)
	go build -o brave.out $(files)

prod: $(files)
	go build -o brave.out -ldflags "-s -w" $(files)

clean:
	rm -f brave.out

.PHONY: all clean
