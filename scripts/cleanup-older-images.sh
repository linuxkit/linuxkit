#!/bin/sh
set -e

##
#
# script to remove all older linuxkit images
# see usage() for usage and functionality
#

usage() {
    cat >&2 <<EOF
$0 [-n] <base> <versions>

Prune any images that start with <base> except the <versions> most recent versions.
[-n] = dry-run, but report what it would remove and leave.

Examples:
  $0 linuxkit 2
        will remove all versions of any image starting with linuxkit except for the 2 most recent of each
        would match "linuxkit/foo" and ALSO "linuxkitprojects/foo"

  $0 linuxkit/ 2
        will remove all versions of any image starting with linuxkit/ except for the 2 most recent of each
        would match "linuxkit/foo" but NOT "linuxkitprojects/foo"

  $0 linuxkit/sshd 3
        will remove all versions of linuxkit/sshd except for the 3 most recent

EOF
}


# backwards compatibility
dryrun=false
if [ "$1" = "-n" ]; then
  dryrun=true
  set -- "$2" "$3"
fi

# sufficient arguments
if [ $# -ne 2 ] ; then
    usage
    exit 1
fi

imagebase="$1"
versions="$2"
unversions=$(( $versions + 1))

# make sure the imagebase is good
case "$imagebase" in
  # has a slash in it
  */*)
    testimage="$imagebase*"
    ;;
    # no slash in it
  *)
    testimage="$imagebase*/*"
    ;;
esac

# find all of our images
IMAGELIST=$(docker image ls --filter=reference="$testimage"':*' --format '{{.Repository}} {{.Tag}} {{.CreatedAt}}')

# find unique names for all images
uniqueimages=$(echo "$IMAGELIST" | awk '{print $1}' | sort | uniq)

# now go through each image, find the list
for img in $uniqueimages; do
  # get the unique list of each, and sort by date
  sortedlist=$(echo "$IMAGELIST" | grep -w $img | sort -k 3,3 -r)
  # now split
  tokeep=$(echo "$sortedlist" | head -$versions | awk '{printf "%s:%s\t",$1,$2; $1=$2=""; print $0}')
  todelete=$(echo "$sortedlist" | tail -n +$unversions | awk '{print $1":"$2}')
  todeletedates=$(echo "$sortedlist" | tail -n +$unversions | awk '{printf "%s:%s\t",$1,$2; $1=$2=""; print $0}')

  echo "$tokeep" | while read i; do
    echo "KEEP\t$i"
  done
  if [ -n "$todeletedates" ]; then
    echo "$todeletedates" | while read i; do
      echo "DELETE\t$i"
    done
    if ! "$dryrun"; then
      docker image rm $todelete
    fi
  fi
done
