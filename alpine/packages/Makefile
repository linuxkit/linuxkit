all:
	$(MAKE) -C 9pudc
	$(MAKE) -C 9pudfuse
	$(MAKE) -C mdnstool
	$(MAKE) -C docker
	$(MAKE) -C hupper
	$(MAKE) -C hvtools

arm:
	$(MAKE) -C 9pudc ARCH=arm
	$(MAKE) -C 9pudfuse ARCH=arm
	$(MAKE) -C mdnstool ARCH=arm
	$(MAKE) -C docker ARCH=arm
	$(MAKE) -C hupper ARCH=arm

clean:
	$(MAKE) -C 9pudc clean
	$(MAKE) -C 9pudfuse clean
	$(MAKE) -C mdnstool clean
	$(MAKE) -C docker clean
	$(MAKE) -C hupper clean
	$(MAKE) -C hvtools clean
