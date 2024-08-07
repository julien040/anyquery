
files := $(wildcard *.go)

all: $(files)
	go build -o vercel.out $(files)

prod: $(files)
	go build -o vercel.out -ldflags "-s -w" $(files)

clean:
	rm -f vercel.out

.PHONY: all clean
