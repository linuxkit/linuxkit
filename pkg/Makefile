DIRS = $(dir $(shell find . -maxdepth 2 -mindepth 2 -type f -name build.yml | sort))
.PHONY: push force-push build forcebuild show-tag clean

OPTIONS ?=

PUSHOPTIONS =

ifneq ($(LK_RELEASE),)
PUSHOPTIONS += -release $(LK_RELEASE)
endif

push:
	linuxkit pkg push $(OPTIONS) $(PUSHOPTIONS) $(DIRS)

forcepush:
	linuxkit pkg push $(OPTIONS) $(PUSHOPTIONS) --force $(DIRS)

build:
	linuxkit pkg build $(OPTIONS) $(DIRS)

forcebuild:
	linuxkit pkg build $(OPTIONS) --force $(DIRS)

show-tag:
	linuxkit pkg show-tag $(OPTIONS) $(DIRS)

clean:
