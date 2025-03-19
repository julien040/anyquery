FILES=$(*.go)

build: $(FILES)
	go build -o server.out $(FILES)

build-linux: $(FILES)
	GOOS=linux GOARCH=amd64 go build -o server.linux.out $(FILES)

run: build
	./server.out

clean:
	rm -f server.out server.linux.out

.PHONY: build build-linux run clean