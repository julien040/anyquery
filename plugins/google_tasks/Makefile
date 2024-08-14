
files := $(wildcard *.go)

all: $(files)
	go build -o google_tasks.out $(files)

prod: $(files)
	go build -o google_tasks.out -ldflags "-s -w" $(files)

clean:
	rm -f google_tasks.out

.PHONY: all clean
