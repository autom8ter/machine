.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "Makefile Commands:"
	@echo "----------------------------------------------------------------"
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
	@echo "----------------------------------------------------------------"

.PHONY: proto
proto: ## regenerate gRPC code
	@echo "generating protobuf code..."
	@rm -rf gen
	@docker run -v `pwd`:/tmp colemanword/prototool:latest prototool generate
	@go fmt ./...