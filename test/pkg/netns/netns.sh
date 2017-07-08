#! /bin/sh

# Initialise the Random Number Seed with the PID
RANDOM=$$

# defaults
ARG_TYPE=veth
ARG_PROTO=tcp
ARG_IP=4
ARG_CONN=1
ARG_ITER=10
ARG_LEN=20
ARG_REUSE=0

usage() {
    echo "Stress test for network namespace using iperf"
    echo " -t  loopback|veth  Type of test [$ARG_TYPE]"
    echo " -p  tcp|udp        Protocol to use [$ARG_PROTO]"
    echo " -ip 4|6            IP version to use [$ARG_IP]"
    echo " -c  <n>            Number of concurrent connections in ns [$ARG_CONN]"
    echo " -i  <n>            Number of iterations [$ARG_ITER]"
    echo " -l  <n>            Maximum length of test before killing [$ARG_LEN]"
    echo " -r                 Re-use network namespace name"
}

# parse arguments
while [[ $# -gt 0 ]]; do
    key="$1"

    case $key in
        -t)
            ARG_TYPE="$2"
            shift
            ;;
        -p)
            ARG_PROTO="$2"
            shift
            ;;
        -ip)
            ARG_IP="$2"
            shift
            ;;
        -c)
            ARG_CONN="$2"
            shift
            ;;
        -i)
            ARG_ITER="$2"
            shift
            ;;
        -l)
            ARG_LEN="$2"
            shift
            ;;
        -r)
            ARG_REUSE=1
            ;;
        *)
            usage
            exit 1
            ;;
    esac
    shift
done

echo "PID=$$"

# Kill a random bit (client, server, network namespace, device) first
# before cleaning up
kill_all() {
    ns=$1
    pid_client=$2
    pid_server=$3
    host_dev=$4
    ns_dev=$5

    R=$(($RANDOM%$#))
    case $R in
        0)
            echo "$ns: Remove namespace first"
            ip netns del "$ns" > /dev/nukk 2>&1 || true
            ;;
        1)
            echo "$ns: Kill client processes first"
            kill "$pid_client"  > /dev/null 2>&1 || true
            ;;
        2)
            echo "$ns: Kill server process first"
            kill "$pid_server"  > /dev/null 2>&1 || true
            ;;
        3)
            echo "$ns: Remove host netdev first"
            ip link del "$host_dev"
            ;;
        4)
            echo "$ns: Remove netns netdev first"
            ip netns exec "$ns" ip link del "$ns_dev"
            ;;
    esac
    kill "$pid_client" > /dev/null 2>&1 || true
    kill "$pid_server" > /dev/null 2>&1 || true
    ip netns del "$ns" > /dev/null 2>&1 || true
    [ "$host_dev"x != x ] && (ip link del "$host_dev" > /dev/null 2>&1 || true) || true
    [ "$ns_dev"x != x ] && (ip netns exec "$ns" ip link del "$ns_dev" > /dev/null 2>&1 || true) || true
}

# Run sock_stress in loopback mode in a network namespace
loopback_run() {
    id=$1 # unique ID for this run, used to create namespace
    ip=$2 # 4 or 6 to select IP version to use
    pr=$3 # protocol tcp or udp

    # Use our PID as the ID to get unique namespaces
    if [ "$ARG_REUSE" = "1" ]; then
        ns="ns_$$"
    else
        ns="ns_$$_$id"
    fi
    ip netns add "$ns"
    ip netns exec "$ns" ip link set lo up

    ip netns exec "$ns" iperf3 -s --logfile /dev/null &
    pid_server=$!
    sleep 1
    [ "$pr" = "udp" ] && o="-u"
    ip netns exec "$ns" iperf3 -"$ip" "$o" -P "$ARG_CONN" -c localhost -t 10000 -i 20 &
    pid_client=$!

    # wait for a while before killing processes
    sleep $(((RANDOM % $ARG_LEN )+1))

    kill_all "$ns" "$pid_client" "$pid_server"
}

# Run sock_stress in with the client in a namespace
veth_run() {
    id=$1 # unique ID for this run, used to create namespace
    ip=$2 # 4 or 6 to select IP version to use
    pr=$3 # tcp or udp

    # Use our PID as the ID to get unique namespaces
    if [ "$ARG_REUSE" = "1" ]; then
        ns="ns_$$"
    else
        ns="ns_$$_$id"
    fi
    dev_host="h-$$-$id"
    dev_ns="n-$$-$id"
    ip netns add "$ns"
    ip netns exec "$ns" ip link set lo up

    ip link add "$dev_host" type veth peer name "$dev_ns"
    ip link set "$dev_ns" netns "$ns"

    # derive IP addresses based on PID and $id
    if [ "$ip" = "4" ]; then
        sub0=$(($$%255))
        sub1=$(($id%255))
        mask=24
        ip_host="10.$sub0.$sub1.1"
        ip_ns="10.$sub0.$sub1.2"
    else
        # Make sure IPv6 is enabled on the interface
        echo 0 > /proc/sys/net/ipv6/conf/"$dev_host"/disable_ipv6
        sub0=$(printf "%x" $(($$%65535)))
        sub1=$(printf "%x" $(($id%65535)))
        mask=64
        ip_host="2001:$sub0:$sub1::1"
        ip_ns="2001:$sub0:$sub1::2"
    fi
    ip -"$ip" addr add "$ip_host"/"$mask" dev "$dev_host"
    ip link set "$dev_host" up

    ip netns exec "$ns" ip -"$ip" addr add "$ip_ns"/"$mask" dev "$dev_ns"
    ip netns exec "$ns" ip link set "$dev_ns" up
    sleep 2 # for IPv6 it takes a little while for the link to come up

    ip netns exec "$ns" iperf3 -s --logfile /dev/null &
    pid_server=$!
    sleep 1
    [ "$pr" = "udp" ] && o="-u"
    iperf3 -"$ip" "$o" -P "$ARG_CONN" -c "$ip_ns" -t 10000 -i 20 &
    pid_client=$!

    # wait for a while before killing processes
    sleep $(((RANDOM % $ARG_LEN )+1))

    kill_all "$ns" "$pid_client" "$pid_server" "$dev_host" "$dev_ns"
}

for i in $(seq 1 "$ARG_ITER"); do
    case $ARG_TYPE in
        veth)
            veth_run "$i" "$ARG_IP" "$ARG_PROTO"
            ;;
        loopback)
            loopback_run "$i" "$ARG_IP" "$ARG_PROTO"
    esac
done
