
files := $(wildcard *.go)

all: $(files)
	go build -o google_forms.out $(files)

prod: $(files)
	go build -o google_forms.out -ldflags "-s -w" $(files)

clean:
	rm -f google_forms.out

.PHONY: all clean
