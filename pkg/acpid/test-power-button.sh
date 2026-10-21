#!/bin/sh
# Exercise the packaged handler in an isolated container PID namespace.
# Usage: ./test-power-button.sh IMAGE
set -eu

image=${1:?Usage: test-power-button.sh IMAGE}
fixture=$(mktemp -d)
trap 'rm -rf "$fixture"' EXIT HUP INT TERM
mkdir "$fixture/input"

# 64-bit Linux input_event: zero timeval, EV_KEY, KEY_POWER, press.
printf '\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\000\001\000\164\000\001\000\000\000' > "$fixture/input/event0"

# Keep the default private PID namespace. Never use --pid=host here.
# BusyBox poweroff sends SIGUSR2 to this shell (PID 1), standing in for init.
docker run --rm --network none --read-only --cap-drop ALL \
    --tmpfs /run \
    --mount "type=bind,src=$fixture/input,dst=/dev/input,readonly" \
    --entrypoint /bin/sh "$image" -c '
        set -eu
        trap '\''echo "PASS: power button requested orderly poweroff"; exit 0'\'' USR2
        test -x /etc/acpi/PWRF/00000080
        test "$(stat -c "%u:%g:%a" /etc/acpi/PWRF/00000080)" = "0:0:755"
        /sbin/acpid -f -d &
        for attempt in 1 2 3 4 5 6 7 8 9 10; do
            sleep 1
        done
        echo "FAIL: no orderly poweroff request" >&2
        exit 1
    '
