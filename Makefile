SHELL := /bin/sh

OLCRTC_DIR ?= $(CURDIR)/olcrtc
BUILD_DIR ?= $(CURDIR)/build
DEPLOY_DIR ?= $(CURDIR)/deploy/olcrtc
ANSIBLE_DIR ?= $(CURDIR)/ansible
INVENTORY ?= $(ANSIBLE_DIR)/inventory.yml
PLAYBOOK ?= $(ANSIBLE_DIR)/olcrtc.yml

SOURCE_URL ?= git@github.com:alananisimov/olcrtc.git
SOURCE_BRANCH ?= deploy-tools
UPSTREAM_URL ?= git@github.com:openlibrecommunity/olcrtc.git
UPSTREAM_BRANCH ?= master

GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0
LDFLAGS ?= -s -w

.PHONY: help status source-init source-publish source-pull source-sync-upstream source-status \
	test provision build inventory ping check deploy

help:
	@printf '%s\n' \
		'Targets:' \
		'  status                Show deployment and source git status' \
		'  source-init           Clone source checkout after its branch is published' \
		'  source-publish        Publish the local source branch to your fork once' \
		'  source-pull           Rebase source changes on openlibrecommunity/master' \
		'  test                  Run tests for deployment-related Go packages' \
		'  provision             Generate deploy/olcrtc/state.yaml from desired.yaml' \
		'  build                 Build Linux olcrtc and olcrtc-subd binaries' \
		'  inventory             Create ansible/inventory.yml from the example' \
		'  ping                  Check Ansible connectivity' \
		'  check                 Run deployment in check/diff mode' \
		'  deploy                Build and apply the playbook'

status:
	@git status --short --branch
	@printf '\nsource checkout:\n'
	@git -C "$(OLCRTC_DIR)" status --short --branch

source-init:
	@test ! -e "$(OLCRTC_DIR)" || { printf '%s\n' "$(OLCRTC_DIR) already exists"; exit 1; }
	git clone --branch "$(SOURCE_BRANCH)" "$(SOURCE_URL)" "$(OLCRTC_DIR)"
	@git -C "$(OLCRTC_DIR)" remote get-url upstream >/dev/null 2>&1 || \
		git -C "$(OLCRTC_DIR)" remote add upstream "$(UPSTREAM_URL)"
	git -C "$(OLCRTC_DIR)" submodule update --init --recursive

source-publish:
	git -C "$(OLCRTC_DIR)" push --set-upstream origin "$(SOURCE_BRANCH)"

source-pull:
	git -C "$(OLCRTC_DIR)" pull --rebase "$(UPSTREAM_URL)" "$(UPSTREAM_BRANCH)"
	git -C "$(OLCRTC_DIR)" submodule update --init --recursive

# Compatibility alias for the earlier target name.
source-sync-upstream: source-pull

source-status:
	git -C "$(OLCRTC_DIR)" status --short --branch
	git -C "$(OLCRTC_DIR)" remote -v

test:
	cd "$(OLCRTC_DIR)" && go test ./internal/transport/vp8channel ./internal/subscription ./cmd/olcrtc-provision ./cmd/olcrtc-subd

provision:
	@test -f "$(DEPLOY_DIR)/desired.yaml" || { \
		printf '%s\n' "Missing $(DEPLOY_DIR)/desired.yaml; start from desired-example.yaml"; exit 1; \
	}
	cd "$(OLCRTC_DIR)" && go run ./cmd/olcrtc-provision \
		-config "$(DEPLOY_DIR)/desired.yaml" \
		-state "$(DEPLOY_DIR)/state.yaml"

build:
	@mkdir -p "$(BUILD_DIR)"
	cd "$(OLCRTC_DIR)" && GOOS="$(GOOS)" GOARCH="$(GOARCH)" CGO_ENABLED="$(CGO_ENABLED)" \
		go build -trimpath -ldflags="$(LDFLAGS)" -o "$(BUILD_DIR)/olcrtc-$(GOOS)-$(GOARCH)" ./cmd/olcrtc
	cd "$(OLCRTC_DIR)" && GOOS="$(GOOS)" GOARCH="$(GOARCH)" CGO_ENABLED="$(CGO_ENABLED)" \
		go build -trimpath -ldflags="$(LDFLAGS)" -o "$(BUILD_DIR)/olcrtc-subd-$(GOOS)-$(GOARCH)" ./cmd/olcrtc-subd

inventory:
	@test ! -e "$(INVENTORY)" || { printf '%s\n' "$(INVENTORY) already exists"; exit 1; }
	cp "$(ANSIBLE_DIR)/inventory-example.yml" "$(INVENTORY)"

ping:
	@test -f "$(INVENTORY)" || { printf '%s\n' "Missing $(INVENTORY); run make inventory"; exit 1; }
	ansible all -i "$(INVENTORY)" -m ping

check:
	@test -f "$(INVENTORY)" || { printf '%s\n' "Missing $(INVENTORY); run make inventory"; exit 1; }
	@test -f "$(DEPLOY_DIR)/state.yaml" || { printf '%s\n' "Missing generated state; run make provision"; exit 1; }
	ansible-playbook -i "$(INVENTORY)" "$(PLAYBOOK)" --check --diff

deploy: build
	@test -f "$(INVENTORY)" || { printf '%s\n' "Missing $(INVENTORY); run make inventory"; exit 1; }
	@test -f "$(DEPLOY_DIR)/state.yaml" || { printf '%s\n' "Missing generated state; run make provision"; exit 1; }
	ansible-playbook -i "$(INVENTORY)" "$(PLAYBOOK)"
