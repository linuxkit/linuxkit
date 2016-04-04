all:
	$(MAKE) -C 9pudc OS=linux
	$(MAKE) -C proxy OS=linux
	$(MAKE) -C transfused OS=linux
	$(MAKE) -C mdnstool OS=linux
	$(MAKE) -C hupper OS=linux
	$(MAKE) -C hvtools OS=linux
	$(MAKE) -C docker OS=Linux

arm:
	$(MAKE) -C 9pudc OS=linux ARCH=arm
	$(MAKE) -C proxy OS=linux ARCH=arm
	$(MAKE) -C transfused OS=linux ARCH=arm
	$(MAKE) -C mdnstool OS=linux ARCH=arm
	$(MAKE) -C hupper OS=linux ARCH=arm
	$(MAKE) -C docker OS=Linux ARCH=arm

clean:
	$(MAKE) -C 9pudc clean
	$(MAKE) -C proxy clean
	$(MAKE) -C transfused clean
	$(MAKE) -C mdnstool clean
	$(MAKE) -C docker clean
	$(MAKE) -C hupper clean
	$(MAKE) -C hvtools clean
