#!/bin/sh
set -eu

case "$(uname -m)" in
    aarch64|x86_64) ;;
    *) echo "Unsupported input_event layout" >&2; exit 1 ;;
esac
test -x /etc/acpi/PWRF/00000080
test "$(stat -c '%u:%g:%a' /etc/acpi/PWRF/00000080)" = "0:0:755"

# test.yml overrides the package's /dev bind, keeping these devices private.
mkdir -p /dev/input
mkfifo /dev/input/event0
log=$(mktemp)
listener=
writer=
cleanup() {
    for pid in "$listener" "$writer"; do
        [ -z "$pid" ] || kill "$pid" 2>/dev/null || true
    done
    for pid in "$listener" "$writer"; do
        [ -z "$pid" ] || wait "$pid" 2>/dev/null || true
    done
    cat "$log"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

(
    exec 3> /dev/input/event0
    # 64-bit little-endian input_event: zero timeval, EV_KEY, KEY_POWER, press.
    printf '\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\001\000\164\000\001\000\000\000' >&3
    sleep 30
) &
writer=$!
echo "ACPID_TEST_EVENT_READY"
/sbin/acpid -f -d > "$log" 2>&1 &
listener=$!
for attempt in $(seq 1 20); do
    if grep -q 'acpid: PWRF/00000080' "$log"; then
        # Let the spawned handler signal real init before onboot completes.
        sleep 1
        exit 0
    fi
    sleep 1
done
echo "Power-button event was not dispatched" >&2
exit 1
