#!/bin/sh

for f in /proc/sys/fs/binfmt_misc/qemu*
do
	NAME=$(basename $f)
	MAGIC=$(cat $f | grep '^magic' | sed 's/^magic //' | sed 's/\(..\)/\\x\1/g')
	OFFSET=$(cat $f | grep '^offset' | sed 's/^offset //')
	MASK=$(cat $f | grep '^mask' | sed 's/^mask //' | sed 's/\(..\)/\\x\1/g')
	EXEC="/usr/bin/$f-static"
	FLAGS=$(cat $f | grep '^flags:' | sed 's/^flags: //')

	printf "echo \":${NAME}:M:${OFFSET}:${MAGIC}:${MASK}:${EXEC}:${FLAGS}\" > /proc/sys/fs/binfmt_misc/register\n"
done
