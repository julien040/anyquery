
files := $(wildcard *.go)

all: $(files)
	go build -o typeform.out $(files)

prod: $(files)
	go build -o typeform.out -ldflags "-s -w" $(files)

clean:
	rm -f typeform.out

.PHONY: all clean
