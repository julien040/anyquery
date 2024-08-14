
files := $(wildcard *.go)

all: $(files)
	go build -o google_contacts.out $(files)

prod: $(files)
	go build -o google_contacts.out -ldflags "-s -w" $(files)

clean:
	rm -f google_contacts.out

.PHONY: all clean
