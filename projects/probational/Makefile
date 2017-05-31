GO_COMPILE=linuxkit/go-compile:3ba94f14de51b73551417e769d122815ec917ee7

.PHONY: gen-probational
gen-probational: mkprobational
	cd .. && ./probational/mkprobational ./probational/projects.yml

mkprobational: Makefile
	tar cf - vendor -C cmd . | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) --package github.com/linuxkit/linuxkit/projects/probational/cmd -o $@ > tmp_probational.tar
	tar xf tmp_probational.tar > $@
	rm tmp_probational.tar
	touch $@

clean:
	-rm tmp_probational.tar probational.yml mkprobational
