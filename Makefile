build:
	go build -ldflags="-s -w" -o toss .

install: build
	mkdir -p $(HOME)/.local/bin
	cp toss $(HOME)/.local/bin/toss

test:
	go test ./...

clean:
	rm -f toss
