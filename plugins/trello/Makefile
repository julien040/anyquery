
files := $(wildcard *.go)

all: $(files)
	go build -o trello.out $(files)

prod: $(files)
	go build -o trello.out -ldflags "-s -w" $(files)

clean:
	rm -f trello.out

.PHONY: all clean
