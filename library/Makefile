DIRS = $(shell find . -type d -depth 1)
.PHONY: clean dirs $(DIRS)

push: $(DIRS)

$(DIRS):
	$(MAKE) -C $@

clean:
	rm -f hash
