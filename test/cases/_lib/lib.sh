#!/bin/sh

# Source the main regression test library if present
[ -f "${RT_LIB}" ] && . "${RT_LIB}"

# Temporary directory for tests to use.
LINUXKIT_TMPDIR="${RT_PROJECT_ROOT}/_tmp"

# The top-level group.sh of the project creates a env.sh file
# containing environment variables for tests. Source it if present.
[ -f "${LINUXKIT_TMPDIR}/env.sh" ] && . "${LINUXKIT_TMPDIR}/env.sh"

# FIXME: Should source the rtf/lib/lib.sh instead
RT_CANCEL=253
