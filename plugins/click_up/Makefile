
files := $(wildcard *.go)

all: $(files)
	go build -o click_up.out $(files)

prod: $(files)
	go build -o click_up.out -ldflags "-s -w" $(files)

clean:
	rm -f click_up.out

.PHONY: all clean
