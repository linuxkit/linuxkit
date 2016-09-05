DIRS = alpine-base alpine-build-c alpine-build-go
.PHONY: clean dirs $(DIRS)

push: $(DIRS)

$(DIRS):
	$(MAKE) -C $@

clean:
	rm -f hash
