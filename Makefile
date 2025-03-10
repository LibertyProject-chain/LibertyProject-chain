# This Makefile is for building the Geth client easily.
# Includes standard and Windows builds.

.PHONY: geth geth-windows

GOBIN = ./build/bin
GORUN = env GO111MODULE=on go

# Build Geth for the current system
geth:
	$(GORUN) run build/ci.go install ./cmd/geth
	@echo "Done building for the current system."
	@echo "Run \"$(GOBIN)/geth\" to launch geth."

