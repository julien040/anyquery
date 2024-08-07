
files := $(wildcard *.go)

all: $(files)
	go build -o todoist.out $(files)

prod: $(files)
	go build -o todoist.out -ldflags "-s -w" $(files)

clean:
	rm -f todoist.out

.PHONY: all clean
