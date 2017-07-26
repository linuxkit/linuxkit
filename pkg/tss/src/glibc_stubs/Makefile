
.PHONY: all clean install

SOLIB = libgetpwent_r.so

DEPS = getpwent_r.o

all: $(SOLIB)

%.o: %.c
	gcc -c -Wall -Werror -fpic ${CFLAGS} $<

$(SOLIB): $(DEPS)
	gcc -shared -o $(SOLIB) $^

install:
	mkdir -p /out/lib
	cp $(SOLIB) /out/lib
