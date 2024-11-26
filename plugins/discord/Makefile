
files := $(wildcard *.go)

all: $(files)
	go build -o discord.out $(files)

prod: $(files)
	go build -o discord.out -ldflags "-s -w" $(files)

clean:
	rm -f discord.out

.PHONY: all clean
