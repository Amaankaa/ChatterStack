.PHONY: fmt air

fmt:
	@echo "Formatting Go sources..."
	@find . -type f -name '*.go' -not -path '*/vendor/*' -not -path '*/tmp/*' -print0 | xargs -0 gofmt -w

air:
	@echo "Starting Air dev server..."
	@air -c air.toml
