#! /bin/sh

# This script creates a container with runc and then runs sock_stress
# (a flexible stress program for UDP/TCP/virtio/Hyper-V/Unix Domain
# sockets). It configures networking either for IPv4 or IPv6 and bind
# mounts a directory for the Unix Domain Socket. After a randomised
# amount of time (maximum set with -t) it kills the sockstress tests
# or the container. This process is repeated several times (-i).
#
# For networking, currently only veth pairs are used. But we plan to
# extent this in the future.
#
# sock_stress supports multiple concurrent connections and sends a
# configurable amount of data over the socket from a client to a
# server, which echoes the data back. By configuring the amount of
# data sent per connection, sock_stress can be used to create a large
# number of short-lived connections (-s). By default a random,
# relatively large amount of data is transferred.
#
# By default the server is run in the container, with the client in
# the parent namespace. the -r option reverses this.

# set -x

# Initialise the Random Number Seed with the PID
RANDOM=$$

# defaults for arguments
ARG_PROTO=tcp
ARG_IP=4
ARG_CONN=1
ARG_SHORT=0
ARG_ITER=20
ARG_TIME=10
ARG_REV=0

usage() {
    echo "Stress test for network namespace using sock_stress"
    echo " -p  tcp|udp|unix   Protocol to use [$ARG_PROTO]"
    echo " -ip 4|6            IP version to use if tcp|udp [$ARG_IP]"
    echo " -c  <n>            Number of concurrent connections [$ARG_CONN]"
    echo " -s                 Use short lived connections (default long)"
    echo " -i  <n>            Number of iterations [$ARG_ITER]"
    echo " -l  <n>            Maximum time of test before killing in s [$ARG_LEN]"
    echo " -r                 Reverse (client in container)"
}

echo "$@"

# parse arguments
while [[ $# -gt 0 ]]; do
    key="$1"

    case $key in
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
        -s)
            ARG_SHORT=1
            ;;
        -i)
            ARG_ITER="$2"
            shift
            ;;
        -l)
            ARG_TIME="$2"
            shift
            ;;
        -r)
            ARG_REV=1
            ;;
        *)
            usage
            exit 1
            ;;
    esac
    shift
done

# Work out the maximum length of a connection. For short connections
# (ie many connections we transfer a max of 4k for long connections
# (lots of data) we transfer up to 8G per connection)
[ "$ARG_SHORT" = "1" ] && MAX_LEN=4096 || MAX_LEN=8388608
[ "$ARG_PROTO" = "unix" ] && ARG_IP=

# Make sure we do IPv6 if asked for
if [ "$ARG_IP" = "6" ]; then
    echo 0 > /proc/sys/net/ipv6/conf/all/disable_ipv6
    echo 0 > /proc/sys/net/ipv6/conf/default/disable_ipv6
    echo 1 > /proc/sys/net/ipv6/conf/all/forwarding
    echo 1 > /proc/sys/net/ipv6/conf/default/forwarding
fi

veth_ipv4() {
    nspid="$1"
    h_ip="$2"
    n_ip="$3"
    mask=24

    # create veth pair and assign the peer to the namespace
    h_dev="h_$nspid"
    n_dev="n_$nspid"
    ip link add "$h_dev" type veth peer name "$n_dev"
    ip link set "$n_dev" netns "$nspid"

    # set up address and up the devices. Host first
    ip addr add "$h_ip"/"$mask" dev "$h_dev"
    ip link set "$h_dev" up

    ip netns exec "$nspid" ip addr add "$n_ip"/"$mask" dev "$n_dev" 
    ip netns exec "$nspid" ip link set lo up
    ip netns exec "$nspid" ip link set "$n_dev" up
    sleep 2 # Wait for link to settle
}

veth_ipv6() {
    nspid="$1"
    h_ip="$2"
    n_ip="$3"
    mask=64

    # create veth pair and assign the peer to the namespace
    h_dev="h_$nspid"
    n_dev="n_$nspid"
    ip link add "$h_dev" type veth peer name "$n_dev"
    ip link set "$n_dev" netns "$nspid"

    # set up address and up the devices. Host first
    ip -6 addr add "$h_ip"/"$mask" dev "$h_dev"
    ip link set "$h_dev" up

    ip netns exec "$nspid" ip -6 addr add "$n_ip"/"$mask" dev "$n_dev" 
    ip netns exec "$nspid" ip link set lo up
    ip netns exec "$nspid" ip link set "$n_dev" up
    sleep 2 # Wait for link to settle
}

PID="$$"
echo "PID=$PID"
D="/test/$PID"
mkdir -p "$D"

##
## Create a runc config.json file based on the template
##
cp /config.template.json "$D"/config.json

# Add a bind mount for unix domain sockets.
BMOUNT="[{\"destination\": \"/data\", \"type\": \"bind\", \"source\": \"$D\", \"options\": [\"rw\", \"rbind\", \"rprivate\"]}]"
jq --argjson args "$BMOUNT" '.mounts |= . + $args' \
       "$D"/config.json  > "$D"/foo.json && mv "$D"/foo.json "$D"/config.json

##
## Run the test $ARG_ITER time
##
for i in $(seq 1 "$ARG_ITER"); do
    # Work out IP addresses
    if [ "$ARG_IP" = "6" ]; then
        sub0=$(($PID%65535)); sub1=$(($i%65535))
        h_ip="2001:$sub0:$sub1::1"; h_ip_addr="[$h_ip]"
        n_ip="2001:$sub0:$sub1::2"; n_ip_addr="[$n_ip]"
    else
        sub0=$(($PID%255)); sub1=$(($i%255))
        h_ip="10.$sub0.$sub1.1"; h_ip_addr="$h_ip"
        n_ip="10.$sub0.$sub1.2"; n_ip_addr="$n_ip"
    fi

    if [ "$ARG_REV" = "0" ]; then
        # Server in container
        if [ "$ARG_PROTO" = "unix" ]; then
            SADDR="/data/stress.sock"
            CADDR="$D/stress.sock"
        else
            SADDR="$n_ip_addr"
            CADDR="$n_ip_addr"
        fi
        C_CMD="[\"/sock_stress\", \"-s\", \"$ARG_PROTO$ARG_IP://$SADDR\"]"
        H_CMD="/rootfs/sock_stress -c $ARG_PROTO$ARG_IP://$CADDR -i 10000000 -l $MAX_LEN -p $ARG_CONN -v 1"
    else
        # Client in container
        if [ "$ARG_PROTO" = "unix" ]; then
            CADDR="/data/stress.sock"
            SADDR="$D/stress.sock"
        else
            SADDR="$h_ip_addr"
            CADDR="$h_ip_addr"
        fi
        C_CMD="[\"/sock_stress\", \"-c\", \"$ARG_PROTO$ARG_IP://$CADDR\", \"-i\", \"10000000\", \"-l\", \"$MAX_LEN\", \"-p\", \"$ARG_CONN\", \"-v\", \"1\"]"
        H_CMD="/rootfs/sock_stress -s $ARG_PROTO$ARG_IP://$SADDR"
    fi

    # Splice container command into json
    jq --argjson args "$C_CMD" '.process.args = $args' \
       "$D"/config.json  > "$D"/foo.json && mv "$D"/foo.json "$D"/config.json

    # Create container, get the namespace ID, and set up symlink for ip utils
    CNAME="c-$PID-$i"
    runc create -b "$D" "$CNAME"
    nspid=$(runc list -f json | jq --arg id "$CNAME" -r '.[] | select(.id==$id) | .pid')
    mkdir -p /var/run/netns && \
        ln -s /proc/"$nspid"/ns/net /var/run/netns/"$nspid"

    # Configure network
    if [ "$ARG_IP" = "6" ]; then
        veth_ipv6 "$nspid" "$h_ip" "$n_ip"
    else
        veth_ipv4 "$nspid" "$h_ip" "$n_ip"
    fi

    # Run
    if [ "$ARG_REV" = "0" ]; then
        runc start "$CNAME"
        sleep 2 # Wait for container to start
        $H_CMD &
        pid_host=$!
    else
        $H_CMD &
        pid_host=$!
        sleep 1 # Wait for server to start
        runc start "$CNAME"
    fi

    # wait for a while before killing processes
    sleep $(((RANDOM % $ARG_TIME )+1))

    R=$(($RANDOM%2))
    case $R in
        0)
            echo "Kill test first"
            kill -9 "$pid_host"
            runc kill "$CNAME"
            runc delete "$CNAME"
            ;;
        1)
            echo "Kill container first"
            runc kill "$CNAME"
            runc delete "$CNAME"
            kill -9 "$pid_host"
            ;;
    esac

    rm /var/run/netns/"$nspid"
done
rm -rf "$D"
