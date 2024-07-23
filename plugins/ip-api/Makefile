
files := $(wildcard *.go)

all: $(files)
	go build -o ip_api.out $(files)

prod: $(files)
	go build -o ip_api.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
