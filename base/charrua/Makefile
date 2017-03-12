OCAML_COMPILE=ocaml-compile
CHARRUA=sbin/charrua
LIBS="logs,mirage-net-fd,ipaddr,charrua-client,astring,cmdliner,fmt.cli,logs.fmt,logs.cli,jsonm,fmt.tty"

$(CHARRUA): charrua.ml
	mkdir -p $(dir $@)
	tar cf - $^ | docker run --rm --net=none --log-driver=none -i $(OCAML_COMPILE) -o $@ --libs $(LIBS) | tar xf -

clean:
	rm -rf hash sbin usr

.DELETE_ON_ERROR:
