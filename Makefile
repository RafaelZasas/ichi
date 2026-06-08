
# Version derived from latest git tag + "-dev" suffix
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")-dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# ldflags matching goreleaser config
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

# Output directory
OUT := ./dist/dev

.PHONY: build
build: ## Build the ichi binary
	@mkdir -p $(OUT)
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o $(OUT)/ichi ./cmd/ichi/
	@echo "Built: $(OUT)/ichi ($(VERSION))"

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'


