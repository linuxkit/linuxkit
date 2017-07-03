#!/bin/sh
set -x

infinite_loop() {
	while true; do
		$@
	done
}

# run getty on all known consoles
start_getty() {
	tty=${1%,*}
	speed=${1#*,}
	securetty="$2"
	line=
	term="linux"
	[ "$speed" = "$1" ] && speed=115200

	case "$tty" in
	ttyS*|ttyAMA*|ttyUSB*|ttyMFD*)
		line="-L"
		term="vt100"
		;;
	tty?)
		line=""
		speed="38400"
		term=""
		;;
	esac

	# are we secure or insecure?
	loginargs=
	if [ "$INSECURE" == "true" ]; then
		loginargs="-a root"
	fi

	if ! grep -q -w "$tty" "$securetty"; then
		echo "$tty" >> "$securetty"
	fi
	# respawn forever
	infinite_loop setsid.getty -w /sbin/agetty $loginargs $line $speed $tty $term &
}

# check if we have /etc/getty.shadow
ROOTSHADOW=/hostroot/etc/getty.shadow
if [ -f $ROOTSHADOW ]; then
	cp $ROOTSHADOW /etc/shadow
	# just in case someone forgot a newline
	echo >> /etc/shadow
fi

# check for scripts that should be added to profile.d
PROFILED=/hostroot/etc/profile.d/
for f in ${PROFILED}*.sh; do
	filename="$(basename ${f})"
	if [ ! -f "/etc/profile.d/${filename}" ]; then
		cp "${f}" /etc/profile.d/
	fi
done

for opt in $(cat /proc/cmdline); do
	case "$opt" in
	console=*)
		start_getty ${opt#console=} /etc/securetty
	esac
done

# wait for all our child process to exit; tini will handle subreaping, if necessary
wait
