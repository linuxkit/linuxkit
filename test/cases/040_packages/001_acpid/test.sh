#!/bin/sh
# SUMMARY: Check that the acpid power-button handler shuts down a LinuxKit guest
# LABELS:
# REPEAT:
set -eu

. "${RT_PROJECT_ROOT}/_lib/lib.sh"

case "$(uname -m)" in
    aarch64|arm64|x86_64) ;;
    *) echo "SKIP: input_event fixture requires amd64 or arm64"; exit "$RT_CANCEL" ;;
esac

cd "$(dirname "$0")"
WORK_DIR=$(mktemp -d)
runner=
watchdog=
stop_vm() {
    # QEMU is a child of the runner and does not receive its signals.
    if [ -s "$WORK_DIR/state/qemu.pid" ]; then
        qemu_pid=$(cat "$WORK_DIR/state/qemu.pid")
        kill -TERM "$qemu_pid" 2>/dev/null || true
    fi
    kill -TERM "$runner" 2>/dev/null || true
    sleep 1
    kill -TERM "$runner" 2>/dev/null || true
}
cleanup() {
    if [ -n "$watchdog" ]; then
        kill "$watchdog" 2>/dev/null || true
        wait "$watchdog" 2>/dev/null || true
    fi
    if [ -n "$runner" ]; then
        stop_vm
        wait "$runner" 2>/dev/null || true
    fi
    rm -rf "$WORK_DIR"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

linuxkit build --format kernel+initrd --dir "$WORK_DIR" --name acpid test.yml
linuxkit run qemu --kernel --networking none --mem 2048 --state "$WORK_DIR/state" "$WORK_DIR/acpid" > "$WORK_DIR/run.log" 2>&1 &
runner=$!
(
    for attempt in $(seq 1 120); do
        kill -0 "$runner" 2>/dev/null || exit 0
        sleep 1
    done
    : > "$WORK_DIR/timed-out"
    stop_vm
) &
watchdog=$!
status=0
wait "$runner" || status=$?
runner=
kill "$watchdog" 2>/dev/null || true
wait "$watchdog" 2>/dev/null || true
watchdog=
cat "$WORK_DIR/run.log"

if [ -e "$WORK_DIR/timed-out" ] || [ "$status" -ne 0 ]; then
    echo "acpid test suite FAILED: guest did not exit normally" >&2
    exit 1
fi
grep -q 'ACPID_TEST_EVENT_READY' "$WORK_DIR/run.log"
grep -q 'acpid: PWRF/00000080' "$WORK_DIR/run.log"
grep -q 'Requesting system poweroff' "$WORK_DIR/run.log"
echo "acpid test suite PASSED"
