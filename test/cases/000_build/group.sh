#!/bin/sh
# SUMMARY: LinuxKit tests for 'linuxkit build'
# LABELS:

group_init() {
    # Group initialisation code goes here
    return 0
}

group_deinit() {
    # Group de-initialisation code goes here
    return 0
}

CMD=$1
case $CMD in
init)
    group_init
    res=$?
    ;;
deinit)
    group_deinit
    res=$?
    ;;
*)
    res=1
    ;;
esac

exit $res
