# Tag: ac075fed7c87e4af30d8490ae0504166cceb0df3
C_COMPILE=mobylinux/c-compile@sha256:0e82d441ce112d638f904a08199c76b022c065a2dbf8908bb366755267d4417f

default: sbin/transfused

DEPS=$(wildcard *.c *.h)

all: sbin/transfused

sbin/transfused: $(DEPS)
	mkdir -p $(dir $@)
	tar cf - $(DEPS) | docker run --rm --net=none --log-driver=none -i $(C_COMPILE) -o $@ | tar xf -

clean:
	rm -rf sbin
