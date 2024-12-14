
files := $(wildcard *.go)

all: $(files)
	go build -o system.out $(files)

prod: $(files)
	go build -o system.out -ldflags "-s -w" $(files)

clean:
	rm -f system.out

.PHONY: all clean
