.PHONY: all prod test release clean

all:
	go build -o livetennis.out .

prod:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o livetennis.out .

test:
	go test -race ./...

release:
	goreleaser build --snapshot --clean

clean:
	rm -f livetennis.out
