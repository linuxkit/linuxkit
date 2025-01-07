
.PHONY: all clean install

SOLIB = libgetpwent_r.so
HEADER = getpwent_r.h

DEPS = getpwent_r.o

all: $(SOLIB)

%.o: %.c
	gcc -c -Wall -Werror -fpic ${CFLAGS} $<

$(SOLIB): $(DEPS)
	gcc -shared -o $(SOLIB) $^

install:
	mkdir -p /out/lib
	cp $(SOLIB) /out/lib
	cp $(HEADER) /usr/include
