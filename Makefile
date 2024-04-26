.PHONY: generate
generate:

.PHONY: build
build:
	./build.sh

.PHONY: test
test:
	./test.sh

.PHONY: clean
clean:
	rm -rf dist

.PHONY: ci-generate
ci-generate: generate

.PHONY: ci-build
ci-build: build

.PHONY: ci-test
ci-test: test
