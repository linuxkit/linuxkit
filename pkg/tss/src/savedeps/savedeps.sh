#!/bin/sh
# find all of my dependencies under $1 and save them to $2
OUTDIR=$1

export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$OUTDIR/lib:$OUTDIR/usr/lib
# find the direct dependencies
DIRECTDEPS=$(for i in $(find $OUTDIR/*bin $OUTDIR/*lib -type f); do ldd $i 2>/dev/null; done | awk '{print $3}' | grep -v '^ldd$' | sort | uniq)
# find the secondary dependencies
SECONDDEPS=$(for i in $DIRECTDEPS; do ldd $i 2>/dev/null; done | awk '{print $3}' | grep -v '^ldd$' | sort | uniq)
# merge together into single unique list, excluding any already in OUTDIR
ALLDEPS=$(echo "$DIRECTDEPS $SECONDDEPS" | sort | uniq | grep -v "^$OUTDIR")


# recursively follows links
copyfile() {
	local infile=${1#/}
	tar cvf - $infile | (cd $OUTDIR ; tar xvf - )
	# if it was a symlink, dereference and copy that
	if [ -L $infile ]; then
		copyfile $(readlink -f $infile)
	fi
}

# we remove the leadink / and then do everything from /
cd /
# save to OUTDIR
mkdir -p $OUTDIR
for infile in $ALLDEPS; do
	if [ ! -e $OUTDIR/$infile ]; then
		# symlinks should be copied but also followed
		copyfile $infile
	fi
done
