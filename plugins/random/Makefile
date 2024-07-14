
files := $(wildcard *.go)

all: $(files)
	go build -o random.out $(files)

prod: $(files)
	go build -o random.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
