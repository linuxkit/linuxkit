#!/bin/sh


KER=$(ls | grep linux-)

for LINUX in $KER
do
	VERSION=$(echo "$LINUX" | sed -e 's/^\(linux-\)//')
	FAMILY=$(echo "$VERSION" | sed -e 's/\.[0-9]*$//').x
	cd $LINUX
    cp /src/split-common-$FAMILY arch/s390/configs/linuxkit_defconfig
	echo "--------- Kernel $VERSION --------------------"
    make defconfig
    
    # Common configuration from x86 and arm
    [ -e /src/split-common-$FAMILY ] || continue
    make linuxkit_defconfig

    cp .config /src/config-$FAMILY-s390x
	echo "----------------------------------------------"
	cd ..
done
