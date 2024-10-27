
files := $(wildcard *.go)

all: $(files)
	go build -o hubspot.out $(files)

prod: $(files)
	go build -o hubspot.out -ldflags "-s -w" $(files)

clean:
	rm -f hubspot.out

.PHONY: all clean
