.PHONY: build install test lint clean release-dry-run completions

build:
	go build -ldflags "-X main.version=dev" -o icu .

install:
	go install -ldflags "-X main.version=dev" .

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f icu

# Generate shell completions into the completions/ directory.
completions: build
	mkdir -p completions
	./icu completion bash  > completions/icu.bash
	./icu completion zsh   > completions/icu.zsh
	./icu completion fish  > completions/icu.fish

# Dry-run goreleaser (requires goreleaser to be installed).
release-dry-run:
	goreleaser release --snapshot --clean
