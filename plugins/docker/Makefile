
files := $(wildcard *.go)

all: $(files)
	go build -o docker.out $(files)

prod: $(files)
	go build -o docker.out -ldflags "-s -w" $(files)

clean:
	rm -f docker.out

.PHONY: all clean
