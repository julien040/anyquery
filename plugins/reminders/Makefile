
files := $(wildcard *.go)

all: $(files)
	go build -o reminder.out $(files)

prod: $(files)
	go build -o reminder.out -ldflags "-s -w" $(files)

clean:
	rm -f notion.out

.PHONY: all clean
