TOOLS_BIN_DIR ?= $(PWD)/bin
CRATE_INSTALL := $(realpath ../../third_party/japaric/trust/crate_install.sh)

MDBOOK := $(TOOLS_BIN_DIR)/mdbook
$(MDBOOK):
	$(CRATE_INSTALL) --git rust-lang/mdBook --tag v0.4.10 --to $(TOOLS_BIN_DIR) --force

MDBOOK_TOC := $(TOOLS_BIN_DIR)/mdbook-toc
$(MDBOOK_TOC):
	$(CRATE_INSTALL) --git badboy/mdbook-toc --tag 0.7.0 --to $(TOOLS_BIN_DIR) --force

DEPS := $(MDBOOK) $(MDBOOK_TOC)

.PHONY: build
build: $(DEPS)
	$(MDBOOK) build

.PHONY: serve
serve: $(DEPS)
	$(MDBOOK) serve
