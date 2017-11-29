.PHONY: default pr all

default: check-deps test 
pr: check-deps test-pr
# TODO: should have a separate target
all: check-deps test-pr

LINUXKIT:=$(shell command -v linuxkit 2> /dev/null)
RTF:=$(shell command -v rtf 2> /dev/null)

.PHONY: check-deps
check-deps:
ifndef LINUXKIT
	$(error "linuxkit binary not found. please install it.")
endif
ifndef RTF
	$(error "rtf is not available. please install it!")
endif

test:
	@rtf -l build -x run

test-pr:
	@rtf -l build -x run
