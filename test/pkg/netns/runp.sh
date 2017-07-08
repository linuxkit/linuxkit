#!/bin/sh

ITER=$1
shift

rm -rf ./logs
mkdir -p ./logs

fail() {
    for f in ./logs/*; do
        echo
        echo "=== $f ==="
        cat $f
    done
    echo
    dmesg
    echo "Test FAILED with $1"
    exit 1
}

ns_before=$(ip netns list | wc -l)

pids=""
for i in $(seq 1 "$ITER"); do
    "$@" > "./logs/$1-$i.log" 2>&1  &
    pid=$!
    pids="$pids $pid"
    echo "Test $i started with PID=$pid"
done

for pid in $pids; do
    wait "$pid"
    [ $? -eq 0 ] || fail "$pid return non-zero"
done

dmesg | grep -q 'Call Trace:' && fail "Kernel backtrace"

# A message like:
# unregister_netdevice: waiting for lo to become free. Usage count = 1
# is somewhat benign as it just waits for the ref count to go to 0. However
# it may become a problem if we have to many of them
nd=$(dmesg | grep -q 'unregister_netdevice' | wc -l)
[ "$nd" -gt 10 ] && fail "unregister_netdevice more than 10 times"

ns_after=$(ip netns list | wc -l)
[ "$ns_before" != "$ns_after" ] && fail "NS leak: $ns_before != $ns_after"

echo "netns test suite PASSED"
