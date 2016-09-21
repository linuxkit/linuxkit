#!/bin/sh

for f in /proc/sys/fs/binfmt_misc/qemu*
do
	NAME="$(basename "$f")"
	MAGIC="$(grep '^magic' "$f" | sed 's/^magic //' | sed 's/\(..\)/\\x\1/g')"
	OFFSET="$(grep '^offset' "$f" | sed 's/^offset //')"
	MASK="$(grep '^mask' "$f" | sed 's/^mask //' | sed 's/\(..\)/\\x\1/g')"
	EXEC="/usr/bin/${NAME}-static"
	FLAGS="$(grep '^flags:' "$f" | sed 's/^flags: //')"

	printf "\techo \":%s:M:%s:%s:%s:%s:%s\" $NAME $OFFSET $MAGIC $MASK $EXEC $FLAGS > /proc/sys/fs/binfmt_misc/register\n"
done
