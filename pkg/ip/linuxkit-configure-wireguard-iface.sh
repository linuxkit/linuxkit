#!/bin/bash
#
# Copyright (C) 2016-2017 Jason A. Donenfeld <Jason@zx2c4.com>. All Rights Reserved.
#
# This file is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License version 2
# as published by the Free Software Foundation.
#
# This file is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this file. If not, see <http://www.gnu.org/licenses/>.

set -e -o pipefail
shopt -s extglob
export LC_ALL=C

SELF="$(readlink -f "${BASH_SOURCE[0]}")"
export PATH="${SELF%/*}:$PATH"

WG_CONFIG=""
INTERFACE=""
ADDRESSES=( )
MTU=""
DNS=( )
CONFIG_FILE=""
PROGRAM="${0##*/}"
ARGS=( "$@" )

parse_options() {
	local interface_section=0 line key value
	CONFIG_FILE="$1"
	[[ -e $CONFIG_FILE ]] || die "\`$CONFIG_FILE' does not exist"
	[[ $CONFIG_FILE =~ /?([a-zA-Z0-9_=+.-]{1,16})\.conf$ ]] || die "The config file must be a valid interface name, followed by .conf"
	((($(stat -c '%#a' "$CONFIG_FILE") & 0007) == 0)) || echo "Warning: \`$CONFIG_FILE' is world accessible" >&2
	INTERFACE="${BASH_REMATCH[1]}"
	shopt -s nocasematch
	while read -r line || [[ -n $line ]]; do
		key="${line%%=*}"; key="${key##*( )}"; key="${key%%*( )}"
		value="${line#*=}"; value="${value##*( )}"; value="${value%%*( )}"
		[[ $key == "["* ]] && interface_section=0
		[[ $key == "[Interface]" ]] && interface_section=1
		if [[ $interface_section -eq 1 ]]; then
			case "$key" in
			Address) ADDRESSES+=( ${value//,/ } ); continue ;;
			MTU) MTU="$value"; continue ;;
			DNS) DNS+=( ${value//,/ } ); continue ;;
			esac
		fi
		WG_CONFIG+="$line"$'\n'
	done < "$CONFIG_FILE"
	shopt -u nocasematch
}

cmd() {
	echo "[#] $*" >&2
	"$@"
}

die() {
	echo "$PROGRAM: $*" >&2
	exit 1
}

up_if() {
	cmd ip link set "$INTERFACE" up
}

add_addr() {
	cmd ip address add "$1" dev "$INTERFACE"
}

set_mtu() {
	local mtu=0 endpoint output
	if [[ -n $MTU ]]; then
		cmd ip link set mtu "$MTU" dev "$INTERFACE"
		return
	fi
	while read -r _ endpoint; do
		[[ $endpoint =~ ^\[?([a-z0-9:.]+)\]?:[0-9]+$ ]] || continue
		output="$(ip route get "${BASH_REMATCH[1]}" || true)"
		[[ ( $output =~ mtu\ ([0-9]+) || ( $output =~ dev\ ([^ ]+) && $(ip link show dev "${BASH_REMATCH[1]}") =~ mtu\ ([0-9]+) ) ) && ${BASH_REMATCH[1]} -gt $mtu ]] && mtu="${BASH_REMATCH[1]}"
	done < <(wg show "$INTERFACE" endpoints)
	if [[ $mtu -eq 0 ]]; then
		read -r output < <(ip route show default || true) || true
		[[ ( $output =~ mtu\ ([0-9]+) || ( $output =~ dev\ ([^ ]+) && $(ip link show dev "${BASH_REMATCH[1]}") =~ mtu\ ([0-9]+) ) ) && ${BASH_REMATCH[1]} -gt $mtu ]] && mtu="${BASH_REMATCH[1]}"
	fi
	[[ $mtu -gt 0 ]] || mtu=1500
	cmd ip link set mtu $(( mtu - 80 )) dev "$INTERFACE"
}

set_dns() {
	[[ ${#DNS[@]} -eq 0 ]] || printf 'nameserver %s\n' "${DNS[@]}" > /etc/resolv.conf
}

add_route() {
	cmd ip route add "$1" dev "$INTERFACE"
}

set_config() {
	cmd wg setconf "$INTERFACE" <(echo "$WG_CONFIG")
}

save_config() {
	local old_umask new_config current_config address
	[[ $(ip -all -brief address show dev "$INTERFACE") =~ ^$INTERFACE\ +\ [A-Z]+\ +(.+)$ ]] || true
	new_config=$'[Interface]\n'
	for address in ${BASH_REMATCH[1]}; do
		new_config+="Address = $address"$'\n'
	done
	if [[ -f /etc/resolv.conf ]]; then
		while read -r address; do
			[[ $address =~ ^nameserver\ ([a-zA-Z0-9_=+:%.-]+)$ ]] && new_config+="DNS = ${BASH_REMATCH[1]}"$'\n'
		done < /etc/resolv.conf
	fi
	[[ -n $MTU && $(ip link show dev "$INTERFACE") =~ mtu\ ([0-9]+) ]] && new_config+="MTU = ${BASH_REMATCH[1]}"$'\n'
	old_umask="$(umask)"
	umask 077
	current_config="$(cmd wg showconf "$INTERFACE")"
	trap 'rm -f "$CONFIG_FILE.tmp"; exit' INT TERM EXIT
	echo "${current_config/\[Interface\]$'\n'/$new_config}" > "$CONFIG_FILE.tmp" || die "Could not write configuration file"
	mv "$CONFIG_FILE.tmp" "$CONFIG_FILE" || die "Could not move configuration file"
	trap - INT TERM EXIT
	umask "$old_umask"
}

cmd_usage() {
	cat >&2 <<-_EOF
	Usage: $PROGRAM { configure | save } CONFIG_FILE

	  CONFIG_FILE is a configuration file, whose filename is the interface name
	  followed by \`.conf'. It is to be readable by wg(8)'s \`setconf' sub-command,
	  with the exception of the following additions to the [Interface] section,
	  which are handled by $PROGRAM:

	  - Address: may be specified one or more times and contains one or more
	    IP addresses (with an optional CIDR mask) to be set for the interface.
	  - DNS: an optional DNS server to use while the device is up.
	  - MTU: an optional MTU for the interface; if unspecified, auto-calculated.
	
	If \`configure' is provided, an existing WireGuard interface is configured using
	this program. If \`save' is provided, an existing WireGuard interface has its
	$PROGRAM configuration written to CONFIG_FILE.
	_EOF
}

cmd_configure() {
	local i
	[[ -z $(ip link show dev "$INTERFACE" 2>/dev/null) ]] && die "\`$INTERFACE' does not exist"
	ip link set "$INTERFACE" down 2>/dev/null || true
	ip -4 address flush dev "$INTERFACE" 2>/dev/null || true
	ip -6 address flush dev "$INTERFACE" 2>/dev/null || true
	ip -4 route flush dev "$INTERFACE" 2>/dev/null || true
	ip -6 route flush dev "$INTERFACE" 2>/dev/null || true
	set_config
	for i in "${ADDRESSES[@]}"; do
		add_addr "$i"
	done
	set_mtu
	up_if
	set_dns
	for i in $(while read -r _ i; do for i in $i; do [[ $i =~ ^[0-9a-z:.]+/[0-9]+$ ]] && echo "$i"; done; done < <(wg show "$INTERFACE" allowed-ips) | sort -nr -k 2 -t /); do
		[[ $(ip route get "$i" 2>/dev/null) == *dev\ $INTERFACE\ * ]] || add_route "$i"
	done
}

cmd_save() {
	[[ -z $(ip link show dev "$INTERFACE" 2>/dev/null) ]] && die "\`$INTERFACE' does not exist"
	save_config
}

if [[ $# -eq 1 && ( $1 == --help || $1 == -h || $1 == help ) ]]; then
	cmd_usage
elif [[ $# -eq 2 && $1 == configure ]]; then
	parse_options "$2"
	cmd_configure
elif [[ $# -eq 2 && $1 == save ]]; then
	parse_options "$2"
	cmd_save
else
	cmd_usage
	exit 1
fi

exit 0
