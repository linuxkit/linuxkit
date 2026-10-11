#!/bin/sh
# SUMMARY: Load the virtio sound module and its dependencies without audio hardware
# LABELS:
# REPEAT:
set -eu

. "${RT_PROJECT_ROOT}/_lib/lib.sh"

# Sound support is enabled only in the aarch64 config in this change.
case "$(uname -m)" in
    aarch64|arm64) ;;
    *) exit "${RT_CANCEL}" ;;
esac

# Resolve the same candidate tag used by the kernel Makefile unless a local override is supplied.
KERNEL_IMAGE="${LINUXKIT_SOUND_KERNEL:-$(make -s -C "${RT_PROJECT_ROOT}/../../kernel" ARCH=arm64 tag-plainkernel-6.12.59)}"
case "${KERNEL_IMAGE}" in
    ''|*[!ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._:/@-]*)
        echo "Invalid LINUXKIT_SOUND_KERNEL image reference" >&2
        exit 1
        ;;
esac

WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/linuxkit-virtio-snd.XXXXXX")"
RUN_ID="${WORK_DIR##*/}"
NAME=virtio-snd
IMAGE_NAME="linuxkit-virtio-snd-test:${RUN_ID}"
cleanup() {
    docker image rm "${IMAGE_NAME}" >/dev/null 2>&1 || true
    rm -rf "${WORK_DIR}"
}
trap cleanup EXIT

docker build -t "${IMAGE_NAME}" .
sed \
    -e "s|linuxkit/kernel:6.12.59|${KERNEL_IMAGE}|" \
    -e "s|linuxkit-virtio-snd-test|${IMAGE_NAME}|" \
    test.yml > "${WORK_DIR}/test.yml"
linuxkit build --docker --format kernel+initrd --dir "${WORK_DIR}" --name "${NAME}" "${WORK_DIR}/test.yml"

RUN_LOG="${WORK_DIR}/run.log"
if linuxkit run "${WORK_DIR}/${NAME}" > "${RUN_LOG}" 2>&1; then
    RUN_STATUS=0
else
    RUN_STATUS=$?
fi
cat "${RUN_LOG}"
if [ "${RUN_STATUS}" -ne 0 ]; then
    echo "linuxkit run failed with status ${RUN_STATUS}" >&2
    exit "${RUN_STATUS}"
fi
grep -q 'Virtio sound module test PASSED' "${RUN_LOG}"
