.PHONY: default pr all

pr: check-deps test
all: check-deps test-all
platforms: check-deps test-platforms
default: pr

MOBY:=$(shell command -v moby 2> /dev/null)
LINUXKIT:=$(shell command -v linuxkit 2> /dev/null)
RTF:=$(shell command -v rtf 2> /dev/null)

.PHONY: check-deps
check-deps:
ifndef MOBY
	$(error "moby is not available. please install it.")
endif
ifndef LINUXKIT
	$(error "linuxkit binary not found. please install it.")
endif
ifndef RTF
	$(error "rtf is not available. please install it!")
endif

test: 
	@rtf -x run

test-pr:
	@rtf -x run

test-all:
	@rtf -l slow,gcp -x run

test-platforms:
	@rtf -l gcp -x run linuxkit.platforms
