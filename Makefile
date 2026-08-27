.PHONY: test check build e2e

test:
	go test ./...

check:
	gofmt -d $$(find . -name '*.go' -not -path './vendor/*')
	@test -z "$$(find . -name '*.go' -not -path './vendor/*' -print0 | xargs -0 wc -l | awk '$$1 > 200 && $$2 != "total"')"
	go vet ./...

build:
	go build ./cmd/url-shortener

e2e:
	./scripts/e2e.sh
