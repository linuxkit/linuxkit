#!/bin/sh

# This script runs multiple runc-net.sh scripts in parallel. It either
# runs identical versions of runc-net.sh with the arguments supplied
# or a number of pre-defined.

ITER=$1
shift

rm -rf ./logs
mkdir -p ./logs

fail() {
    for f in ./logs/*; do
        echo
        echo "=== $f ==="
        cat "$f"
    done
    echo
    dmesg
    echo "Test FAILED with $1"
    exit 1
}

pids=""
case "$ITER" in
    "mix")
        echo "Running a mix /runc-net.sh with servers in containers"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4 -s > ./logs/01.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4 -s > ./logs/02.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6 -s > ./logs/03.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6 -s > ./logs/04.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p unix      -s > ./logs/05.log 2>&1 &
        pid=$!; pids="$pids $pid"

        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4    > ./logs/06.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4    > ./logs/07.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6    > ./logs/08.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6    > ./logs/09.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p unix         > ./logs/10.log 2>&1 &
        pid=$!; pids="$pids $pid"
        ;;
    "mix-reverse")
        echo "Running a mix /runc-net.sh with clients in containers"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4 -s -r > ./logs/01.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4 -s -r > ./logs/02.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6 -s -r > ./logs/03.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6 -s -r > ./logs/04.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p unix      -s -r > ./logs/05.log 2>&1 &
        pid=$!; pids="$pids $pid"

        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4    -r > ./logs/06.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4    -r > ./logs/07.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6    -r > ./logs/08.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6    -r > ./logs/09.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p unix         -r > ./logs/10.log 2>&1 &
        pid=$!; pids="$pids $pid"
        ;;
    "mix-ipv4")
        echo "Running a mix /runc-net.sh tests with IPv4 only"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4 -s    > ./logs/01.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4 -s    > ./logs/02.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4 -s -r > ./logs/03.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4 -s -r > ./logs/04.log 2>&1 &
        pid=$!; pids="$pids $pid"

        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4       > ./logs/05.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4       > ./logs/06.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4    -r > ./logs/07.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4    -r > ./logs/08.log 2>&1 &
        pid=$!; pids="$pids $pid"
        ;;
    "mix-ipv6")
        echo "Running a mix /runc-net.sh tests with IPv6 only"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6 -s    > ./logs/01.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6 -s    > ./logs/02.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6 -s -r > ./logs/03.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6 -s -r > ./logs/04.log 2>&1 &
        pid=$!; pids="$pids $pid"

        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6       > ./logs/05.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6       > ./logs/06.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6    -r > ./logs/07.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6    -r > ./logs/08.log 2>&1 &
        pid=$!; pids="$pids $pid"
        ;;
    "mix-tcp")
        echo "Running a mix /runc-net.sh tests with TCP only"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4 -s    > ./logs/01.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4 -s -r > ./logs/02.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4       > ./logs/03.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 4    -r > ./logs/04.log 2>&1 &
        pid=$!; pids="$pids $pid"

        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6 -s    > ./logs/01.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6 -s -r > ./logs/02.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6       > ./logs/03.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p tcp -ip 6    -r > ./logs/04.log 2>&1 &
        pid=$!; pids="$pids $pid"
        ;;
    "mix-udp")
        echo "Running a mix /runc-net.sh tests with UDP only"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4 -s    > ./logs/01.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4 -s -r > ./logs/02.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4       > ./logs/03.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 4    -r > ./logs/04.log 2>&1 &
        pid=$!; pids="$pids $pid"

        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6 -s    > ./logs/01.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6 -s -r > ./logs/02.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6       > ./logs/03.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p udp -ip 6    -r > ./logs/04.log 2>&1 &
        pid=$!; pids="$pids $pid"
        ;;
    "mix-unix")
        echo "Running a mix /runc-net.sh tests with unix domain sockets"
        /runc-net.sh -i 30 -l 10 -c 5 -p unix      -s    > ./logs/01.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p unix            > ./logs/02.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p unix      -s -r > ./logs/03.log 2>&1 &
        pid=$!; pids="$pids $pid"
        /runc-net.sh -i 30 -l 10 -c 5 -p unix         -r > ./logs/04.log 2>&1 &
        pid=$!; pids="$pids $pid"
        ;;
    *)
        echo "Running $ITER instances of /runc-net.sh $@"

        for i in $(seq 1 "$ITER"); do
            /runc-net.sh $@ > "./logs/$1-$i.log" &
            pid=$!; pids="$pids $pid"
            echo "Test $i started with PID=$pid"
        done
        ;;
esac

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

echo "netns test suite PASSED"
