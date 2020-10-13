.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "Makefile Commands:"
	@echo "----------------------------------------------------------------"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
	@echo "----------------------------------------------------------------"

bench-tests: ## run benchmarks
	go test -v -bench=.

unit-tests: ## run unit tests
	go test -cover -v -coverprofile cover.out .

coverage: ## show test coverage
	go tool cover -func cover.out

gen: ## generate docs
	@go generate ./...
