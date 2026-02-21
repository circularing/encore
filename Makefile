.PHONY: help build build-encore build-git-remote rebuild clean uninstall install-complete reinstall-complete test-dashboard-contracts test-dashboard-e2e test-dashboard-visual dashboard-iso-inventory

# Override at invocation time if needed:
#   make build ENCORE_INSTALL=/custom/path
ENCORE_INSTALL ?= $(HOME)/.encore
BIN_DIR := $(ENCORE_INSTALL)/bin

help:
	@echo "Targets:"
	@echo "  make build          Build encore + git-remote-encore"
	@echo "  make build-encore   Build only encore"
	@echo "  make build-git-remote Build only git-remote-encore"
	@echo "  make rebuild        Rebuild encore + git-remote-encore"
	@echo "  make clean          Non-destructive clean (go build cache)"
	@echo "  make uninstall      Remove encore binaries from $(BIN_DIR)"
	@echo "  make install-complete  Build binaries and sync runtimes into $(ENCORE_INSTALL)"
	@echo "  make reinstall-complete  Remove runtimes then run install-complete"
	@echo "  make dashboard-iso-inventory  Generate local dashboard route/contract inventory"
	@echo "  make test-dashboard-contracts  Run local dashboard backend contract-oriented tests"
	@echo "  make test-dashboard-e2e  Run dashboard Playwright tests if configured"
	@echo "  make test-dashboard-visual  Run dashboard visual regression tests if configured"

build: build-encore build-git-remote

build-encore:
	@mkdir -p "$(BIN_DIR)"
	go build -o "$(BIN_DIR)/encore" ./cli/cmd/encore

build-git-remote:
	@mkdir -p "$(BIN_DIR)"
	go build -o "$(BIN_DIR)/git-remote-encore" ./cli/cmd/git-remote-encore

rebuild: build

clean:
	go clean -cache

uninstall:
	rm -f "$(BIN_DIR)/encore" "$(BIN_DIR)/git-remote-encore"

# Complete local install inspired by the official installer:
# - installs binaries in $(ENCORE_INSTALL)/bin
# - syncs runtimes into $(ENCORE_INSTALL)/runtimes
# - preserves local source-built binaries
install-complete: build
	@mkdir -p "$(ENCORE_INSTALL)/runtimes"
	@rm -rf "$(ENCORE_INSTALL)/runtimes/go" "$(ENCORE_INSTALL)/runtimes/js"
	cp -R "./runtimes/go" "$(ENCORE_INSTALL)/runtimes/go"
	cp -R "./runtimes/js" "$(ENCORE_INSTALL)/runtimes/js"
	@echo "Installed Encore binaries to $(BIN_DIR)"
	@echo "Installed runtimes to $(ENCORE_INSTALL)/runtimes"
	@if [ ! -d "$(ENCORE_INSTALL)/encore-go" ]; then \
		echo "WARN: $(ENCORE_INSTALL)/encore-go is missing."; \
		echo "      Run the official installer once if needed:"; \
		echo "      curl -L https://encore.dev/install.sh | bash"; \
	fi
	@if command -v encore >/dev/null 2>&1; then \
		echo "Run 'encore version' to verify."; \
	else \
		echo "Add to PATH: export ENCORE_INSTALL=$(ENCORE_INSTALL) && export PATH=\$$ENCORE_INSTALL/bin:\$$PATH"; \
	fi

reinstall-complete:
	@rm -rf "$(ENCORE_INSTALL)/runtimes"
	$(MAKE) install-complete

dashboard-iso-inventory:
	./scripts/dashboard/inventory.sh

test-dashboard-contracts:
	go test ./cli/daemon/dash/...

test-dashboard-e2e:
	@if [ -f package.json ] && grep -q '"test:dashboard:e2e"' package.json; then \
		npm run test:dashboard:e2e; \
	else \
		echo "No dashboard e2e script configured yet (expected in Sprint 1)."; \
	fi

test-dashboard-visual:
	@if [ -f package.json ] && grep -q '"test:dashboard:visual"' package.json; then \
		npm run test:dashboard:visual; \
	else \
		echo "No dashboard visual script configured yet (expected in Sprint 1)."; \
	fi

