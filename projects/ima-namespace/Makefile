.PHONY: run
run: ima-namespace-kernel
	../../bin/linuxkit run ima-namespace

ima-namespace-kernel: ima-namespace.yml
	../../bin/moby build ima-namespace

.PHONY: clean
clean:
	-rm *-cmdline *-kernel *.img
