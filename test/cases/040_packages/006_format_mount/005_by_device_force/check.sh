#!/bin/sh

function failed {
	printf "format_force test suite FAILED\n" >&1
	exit 1
}

# sda should have been partitioned and sda1 formatted as ext4
#   command: ["/usr/bin/format", "-verbose", "-type", "ext4", "/dev/sda"]

# sdb should have been partitioned and sdb1 formatted as ext4
#  command: ["/usr/bin/format", "-verbose", "-type", "ext4", "/dev/sdb"]

# sda1 should remain ext4, as the format was not re-forced
#  command: ["/usr/bin/format", "-verbose", "-type", "xfs", "/dev/sda"]

# sdb should have been re-partitioned, with sdb1 now formatted as xfs due to -force flag
#  command: ["/usr/bin/format", "-verbose", "-force", "-type", "xfs", "/dev/sdb"]

ATTEMPT=0

while true; do
  ATTEMPT=$((ATTEMPT+1))

  echo "=== forcing device discovery (attempt ${ATTEMPT}) ==="
  mdev -s

  echo "=== /dev list (attempt ${ATTEMPT}) ==="
  ls -al /dev

  if [ -b /dev/sda1 ] && [ -b /dev/sdb1 ]; then
    echo 'Found /dev/sda1 and /dev/sdb1 block devices'
    break
  fi

  if [ $ATTEMPT -ge 10 ]; then
    echo "Did not detect /dev/sda1 nor /dev/sdb1 in ${ATTEMPT} attempts"
    failed
  fi

  sleep 1
done

echo "=== /dev/sda1 ==="
blkid -o export /dev/sda1
echo "=== /dev/sdb1 ==="
blkid -o export /dev/sdb1

echo "=== /dev/sda1 test ==="
blkid -o export /dev/sda1 | grep -Fq 'TYPE=ext4' || failed
echo "=== /dev/sdb1 test ==="
blkid -o export /dev/sdb1 | grep -Fq 'TYPE=xfs' || failed

printf "format_force test suite PASSED\n" >&1
