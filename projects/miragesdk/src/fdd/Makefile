.PHONY: all clean

all:
	jbuilder build --dev

clean:
	jbuilder clean

### FIXME: use Makefile template

IMAGE=samoht/fdd

tag:
	docker build -t $(IMAGE) . && docker push $(IMAGE)
