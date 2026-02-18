build:
	go build -ldflags="-s -w" -o toss .

install: build
	mkdir -p $(HOME)/.local/bin
	cp toss $(HOME)/.local/bin/toss
	mkdir -p $(HOME)/.local/share/man/man1
	cp man/toss.1 $(HOME)/.local/share/man/man1/toss.1
	@echo "Installed to $(HOME)/.local/bin/toss"
	@echo "Make sure $(HOME)/.local/bin is on your PATH (re-login or source your shell profile if not)."

test:
	go test ./...

clean:
	rm -f toss
