.PHONY: build serve clean

build:
	go run ./cmd

serve:
	go run ./cmd -serve 8081

clean:
	rm -rf dist
