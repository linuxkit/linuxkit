# Makefile for llmnrd
#
# Copyright (C) 2014-2015 Tobias Klauser <tklauser@distanz.ch>

VERSION = 0.1-rc1

# llmnrd binary
D_P 	= llmnrd
D_OBJS	= llmnr.o iface.o socket.o util.o llmnrd.o
D_LIBS	= -lpthread

# llmnr-query binary
Q_P 	= llmnr-query
Q_OBJS	= util.o llmnr-query.o
Q_LIBS	=

CC	= $(CROSS_COMPILE)gcc
INSTALL	= install

CFLAGS	?= -W -Wall -O2
LDFLAGS	?=

ifeq ($(shell git rev-parse > /dev/null 2>&1; echo $$?), 0)
  GIT_VERSION = "(git id $(shell git describe --always))"
else
  GIT_VERSION =
endif

CFLAGS	+= -DVERSION_STRING=\"v$(VERSION)\" -DGIT_VERSION=\"$(GIT_VERSION)\"

ifeq ($(DEBUG), 1)
  CFLAGS += -g -DDEBUG
endif

Q	?= @
CCQ	= $(Q)echo "  CC $<" && $(CC)
LDQ	= $(Q)echo "  LD $@" && $(CC)

prefix	?= /usr/local

BINDIR	= $(prefix)/bin
SBINDIR	= $(prefix)/sbin
DESTDIR	=

all: $(D_P) $(Q_P)

$(D_P): $(D_OBJS)
	$(LDQ) $(LDFLAGS) -o $@ $(D_OBJS) $(D_LIBS)

$(Q_P): $(Q_OBJS)
	$(LDQ) $(LDFLAGS) -o $@ $(Q_OBJS) $(Q_LIBS)

%.o: %.c %.h
	$(CCQ) $(CFLAGS) -o $@ -c $<

%.o: %.c
	$(CCQ) $(CFLAGS) -o $@ -c $<

install_$(D_P): $(D_P)
	@echo "  INSTALL $(D_P)"
	@$(INSTALL) -d -m 755 $(DESTDIR)$(SBINDIR)
	@$(INSTALL) -m 755 $(D_P) $(DESTDIR)$(SBINDIR)/$(D_P)

install_$(Q_P): $(Q_P)
	@echo "  INSTALL $(Q_P)"
	@$(INSTALL) -d -m 755 $(DESTDIR)$(BINDIR)
	@$(INSTALL) -m 755 $(Q_P) $(DESTDIR)$(BINDIR)/$(Q_P)

install: install_$(D_P) install_$(Q_P)

clean:
	@echo "  CLEAN"
	@rm -f $(D_OBJS) $(D_P)
	@rm -f $(Q_OBJS) $(Q_P)
