#!/bin/sh
# NAME: linuxkit
# SUMMARY: LinuxKit Regression Tests
# LABELS:

# Source libraries. Uncomment if needed/defined
# . "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

group_init() {
    # Group initialisation code goes here
    [ -r "${LINUXKIT_TMPDIR}" ] && rm -rf "${LINUXKIT_TMPDIR}"
    mkdir "${LINUXKIT_TMPDIR}"
    [ -r "${LINUXKIT_ARTIFACTS_DIR}" ] && rm -rf "${LINUXKIT_ARTIFACTS_DIR}"
    mkdir "${LINUXKIT_ARTIFACTS_DIR}"
    echo "export LINUXKIT_EXAMPLES_DIR=${RT_PROJECT_ROOT}/../../examples" >> "${LINUXKIT_TMPDIR}/env.sh"
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

