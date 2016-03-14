all:
	$(MAKE) -C 9pudc OS=linux
	$(MAKE) -C transfused OS=linux
	$(MAKE) -C mdnstool OS=linux
	$(MAKE) -C docker OS=linux
	$(MAKE) -C hupper OS=linux
	$(MAKE) -C hvtools OS=linux

arm:
	$(MAKE) -C 9pudc ARCH=arm
	$(MAKE) -C transfused ARCH=arm
	$(MAKE) -C mdnstool ARCH=arm
	$(MAKE) -C docker ARCH=arm
	$(MAKE) -C hupper ARCH=arm

clean:
	$(MAKE) -C 9pudc clean
	$(MAKE) -C transfused clean
	$(MAKE) -C mdnstool clean
	$(MAKE) -C docker clean
	$(MAKE) -C hupper clean
	$(MAKE) -C hvtools clean
