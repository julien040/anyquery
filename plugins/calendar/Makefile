
files := $(wildcard *.go)

all: $(files)
	go build -o icalendar.out $(files)

prod: $(files)
	go build -o icalendar.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
