#!/bin/sh

errs=0

function lint {
  full_tag="$1"
  img=${1%%:*}
  echo "linting ${full_tag}..."
  while IFS="" read -r m
  do
    if test "${m#*$full_tag}" == "$m"; then
      printf "ERROR: $m\n\n"
      errs=$((errs+1))
    fi
  done < <(grep -R "${img}:" ./cases)
}

while IFS="" read -r p || [ -n "$p" ]
do
  lint $p
done < latest-tags

if [ $errs -gt 0 ]; then
  echo "Linter found $errs errors"
  exit 1
fi
