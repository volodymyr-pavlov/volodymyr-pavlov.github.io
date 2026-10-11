.PHONY: build serve test vet fmt check clean

build:
	go run ./app/cmd/gen

serve:
	go run ./app/cmd/gen -serve 9090

test:
	go test ./...

vet:
	go vet ./...

# Fails, listing the files, when any is not gofmt-formatted.
fmt:
	@unformatted="$$(gofmt -l app)"; \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi

# What CI runs before it builds.
check: fmt vet test

clean:
	rm -rf .dist
