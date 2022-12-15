#!/bin/sh

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

	# did we already process this tty?
	if $(echo "${PROCESSEDTTY}" | grep -q -w "$tty"); then
		echo "getty: already processed tty for $tty, not starting twice" | tee /dev/console
		return
	fi
	# now indicate that we are processing it
	PROCESSEDTTY="${PROCESSEDTTY} ${tty}"

	# does the device even exist?
	if [ ! -c /dev/$tty ]; then
		echo "getty: cmdline has console=$tty but /dev/$tty is not a character device; not starting getty for $tty" | tee /dev/console
		return
	fi

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
		# we could not find the tty in securetty, so start a getty but warn that root login will not work
		echo "getty: cmdline has console=$tty but does not exist in $securetty; will not be able to log in as root on this tty $tty." | tee /dev/$tty
	fi
	# respawn forever
	echo "getty: starting getty for $tty"  | tee /dev/$tty
	infinite_loop setsid.getty -w /sbin/agetty $loginargs $line $speed $tty $term &
}


# check if we are namespaced, and, if so, indicate in the PS1
if [ -z "$INITGETTY" ]; then
	cat >/etc/profile.d/namespace.sh <<"EOF"
export PS1="(ns: getty) $PS1"
EOF
fi

PROCESSEDTTY=

# check if we have /etc/getty.shadow
ROOTSHADOW=/hostroot/etc/getty.shadow
if [ -f $ROOTSHADOW ]; then
	cp $ROOTSHADOW /etc/shadow
	# just in case someone forgot a newline
	echo >> /etc/shadow
fi

for opt in $(cat /proc/cmdline); do
	case "$opt" in
	console=*)
		start_getty ${opt#console=} /etc/securetty
	esac
done

# if we are in a container (not in root init) wait for all our child process to exit; tini will handle subreaping, if necessary
[ -n "$INITGETTY" ] || wait
