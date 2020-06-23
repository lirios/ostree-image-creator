# SPDX-FileCopyrightText: 2020 Pier Luigi Fiorini <pierluigi.fiorini@gmail.com>
#
# SPDX-License-Identifier: CC0-1.0

TAGS :=
LDFLAGS := -w -s
GOFLAGS :=

DESTDIR :=
PREFIX := /usr/local
BINDIR := $(CURDIR)/bin
BINNAME ?= oic

C_SRC := $(shell find . -type f -name "*.[ch]")
GO_SRC := $(shell find . -type f -name "*.go")

.PHONY: all
all: build

.PHONY: build
build: $(BINDIR)/$(BINNAME)

$(BINDIR)/$(BINNAME): $(GO_SRC) $(C_SRC)
	(cd cmd/oic && GO111MODULE=on go build $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $@)

.PHONY: install
install: $(BINDIR)/$(BINNAME)
	install -Dm755 $< $(DESTDIR)$(PREFIX)/bin/$(BINNAME)

.PHONY: clean
clean:
	@rm -rf $(BINDIR)

.PHONY: test
test:
	GO111MODULE=on go test $(GOFLAGS) -run . ./...

.PHONY: format
format:
	GO111MODULE=on gofmt -w .
	clang-format -i $(C_SRC)
