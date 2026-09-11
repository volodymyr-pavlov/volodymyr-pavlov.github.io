.PHONY: build serve clean

# .env (git-ignored) holds AUTHOR_EMAIL=... so the address stays out of the repository.
-include .env
export AUTHOR_EMAIL

build:
	go run ./cmd

serve:
	go run ./cmd -serve 8081

clean:
	rm -rf dist
