GO ?= go
DOMAIN=incus-compose
POFILES=$(wildcard po/*.po)
MOFILES=$(patsubst %.po,%.mo,$(POFILES))
LINGUAS=$(basename $(POFILES))
POTFILE=po/$(DOMAIN).pot
VERSION=$(or ${CUSTOM_VERSION},$(shell grep "var Version" internal/version/flex.go | cut -d'"' -f2))
ARCHIVE=incus-compose-$(VERSION).tar
HASH := \#
TAG_SQLITE3=$(shell printf "$(HASH)include <cowsql.h>\nvoid main(){cowsql_node_id n = 1;}" | $(CC) ${CGO_CFLAGS} -o /dev/null -xc - >/dev/null 2>&1 && echo "libsqlite3")
GOPATH ?= $(shell $(GO) env GOPATH)
CGO_LDFLAGS_ALLOW ?= (-Wl,-wrap,pthread_create)|(-Wl,-z,now)


ifneq "$(wildcard vendor)" ""
	RAFT_PATH=$(CURDIR)/vendor/raft
	COWSQL_PATH=$(CURDIR)/vendor/cowsql
else
	RAFT_PATH=$(GOPATH)/deps/raft
	COWSQL_PATH=$(GOPATH)/deps/cowsql
endif


.PHONY: default
default: client


.PHONY: client
client:
	$(GO) install -v -tags "$(TAG_SQLITE3)" $(DEBUG) ./cmd/incus-compose
	@echo "Incus compose built successfully"



.PHONY: update-gomod
update-gomod:
ifneq "$(INCUS_OFFLINE)" ""
	@echo "The update-gomod target cannot be run in offline mode."
	exit 1
endif
	$(GO) get -t -v -d -u ./...
	$(GO) mod tidy --go=1.233
	$(GO) get toolchain@none

	@echo "Dependencies updated"



.PHONY: check
check: default
ifeq "$(INCUS_OFFLINE)" ""
	(cd / ; $(GO) install -v -x github.com/rogpeppe/godeps@latest)
	(cd / ; $(GO) install -v -x github.com/tsenart/deadcode@latest)
	(cd / ; $(GO) install -v -x golang.org/x/lint/golint@latest)
endif
	CGO_LDFLAGS_ALLOW="$(CGO_LDFLAGS_ALLOW)" $(GO) test -v -tags "$(TAG_SQLITE3)" $(DEBUG) ./...
	cd test && ./main.sh

.PHONY: dist
dist: 
	# Cleanup
	rm -Rf $(ARCHIVE).xz

	# Create build dir
	$(eval TMP := $(shell mktemp -d))
	git archive --prefix=incus-compose-$(VERSION)/ HEAD | tar -x -C $(TMP)
	git show-ref HEAD | cut -d' ' -f1 > $(TMP)/incus-compose-$(VERSION)/.gitref

	# Download dependencies
	(cd $(TMP)/incus-compose-$(VERSION) ; $(GO) mod vendor)


	# Assemble tarball
	tar --exclude-vcs -C $(TMP) -Jcf $(ARCHIVE).xz incus-compose-$(VERSION)/

	# Cleanup
	rm -Rf $(TMP)

.PHONY: i18n
i18n: update-pot update-po

po/%.mo: po/%.po
	msgfmt --statistics -o $@ $<

po/%.po: po/$(DOMAIN).pot
	msgmerge -U po/$*.po po/$(DOMAIN).pot

.PHONY: update-po
update-po:
	set -eu; \
	for lang in $(LINGUAS); do\
	    msgmerge --backup=none -U $$lang.po po/$(DOMAIN).pot; \
	done

.PHONY: update-pot
update-pot:
ifeq "$(INCUS_OFFLINE)" ""
	(cd / ; $(GO) install -v -x github.com/snapcore/snapd/i18n/xgettext-go@2.57.1)
endif
	xgettext-go -o po/$(DOMAIN).pot --add-comments-tag=TRANSLATORS: --sort-output --package-name=$(DOMAIN) --msgid-bugs-address=lxc-devel@lists.linuxcontainers.org --keyword=i18n.G --keyword-plural=i18n.NG cmd/incus-compose/*.go

.PHONY: build-mo
build-mo: $(MOFILES)

.PHONY: static-analysis
static-analysis:
ifeq ($(shell command -v go-licenses),)
	(cd / ; $(GO) install -v -x github.com/google/go-licenses@latest)
endif
ifeq ($(shell command -v golangci-lint),)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$($(GO) env GOPATH)/bin
endif
ifeq ($(shell command -v shellcheck),)
	echo "Please install shellcheck"
	exit 1
else
endif
ifeq ($(shell command -v flake8),)
	echo "Please install flake8"
	exit 1
endif
	flake8 test/deps/import-busybox
	shellcheck --shell sh test/*.sh test/includes/*.sh test/suites/*.sh test/backends/*.sh test/lint/*.sh
	shellcheck test/extras/*.sh
	run-parts $(shell run-parts -V >/dev/null 2>&1 && echo -n "--verbose --exit-on-error --regex '.sh'") test/lint

.PHONY: staticcheck
staticcheck:
ifeq ($(shell command -v staticcheck),)
	(cd / ; $(GO) install -v -x honnef.co/go/tools/cmd/staticcheck@latest)
endif
	# To get advance notice of deprecated function usage, consider running:
	#   sed -i 's/^go 1\.[0-9]\+$/go 1.18/' go.mod
	# before 'make staticcheck'.

	# Run staticcheck against all the dirs containing Go files.
	staticcheck $$(git ls-files *.go | sed 's|^|./|; s|/[^/]\+\.go$$||' | sort -u)
