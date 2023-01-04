DIRS = $(dir $(shell find . -maxdepth 2 -mindepth 2 -type f -name build.yml | sort))
.PHONY: push forcepush build forcebuild show-tag clean

OPTIONS ?=

push:
	linuxkit pkg push $(OPTIONS) $(DIRS)

forcepush:
	linuxkit pkg push $(OPTIONS) --force $(DIRS)

build:
	linuxkit pkg build $(OPTIONS) $(DIRS)

forcebuild:
	linuxkit pkg build $(OPTIONS) --force $(DIRS)

show-tag:
	linuxkit pkg show-tag $(OPTIONS) $(DIRS)

clean:
