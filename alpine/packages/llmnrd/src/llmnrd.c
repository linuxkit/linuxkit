/*
 * llmnrd -- LLMNR (RFC 4705) responder daemon.
 *
 * Copyright (C) 2014-2015 Tobias Klauser <tklauser@distanz.ch>
 *
 * This file is part of llmnrd.
 *
 * llmnrd is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, version 2 of the License.
 *
 * llmnrd is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with llmnrd.  If not, see <http://www.gnu.org/licenses/>.
 */

#include <errno.h>
#include <getopt.h>
#include <signal.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include <sys/ioctl.h>

#include "compiler.h"
#include "log.h"
#include "util.h"

#include "iface.h"
#include "llmnr.h"
#include "llmnr-packet.h"

static const char *short_opts = "H:p:6dhV";
static const struct option long_opts[] = {
	{ "hostname",	required_argument,	NULL, 'H' },
	{ "port",	required_argument,	NULL, 'p' },
	{ "ipv6",	no_argument,		NULL, '6' },
	{ "daemonize",	no_argument,		NULL, 'd' },
	{ "help",	no_argument,		NULL, 'h' },
	{ "version",	no_argument,		NULL, 'V' },
	{ NULL,		0,			NULL, 0 },
};

static void __noreturn usage_and_exit(int status)
{
	fprintf(stdout, "Usage: llmnrd [OPTIONS]\n"
			"Options:\n"
			"  -H, --hostname NAME  set hostname to respond with (default: system hostname)\n"
			"  -p, --port NUM       set port number to listen on (default: %d)\n"
			"  -6, --ipv6           enable LLMNR name resolution over IPv6\n"
			"  -d, --daemonize      run as daemon in the background\n"
			"  -h, --help           show this help and exit\n"
			"  -V, --version        show version information and exit\n",
			LLMNR_UDP_PORT);
	exit(status);
}

static void __noreturn version_and_exit(void)
{
	fprintf(stdout, "llmnrd %s %s\n"
			"Copyright (C) 2014-2015 Tobias Klauser <tklauser@distanz.ch>\n"
			"Licensed under the GNU General Public License, version 2\n",
			VERSION_STRING, GIT_VERSION);
	exit(EXIT_SUCCESS);
}

static void signal_handler(int sig)
{
	switch (sig) {
	case SIGINT:
	case SIGQUIT:
	case SIGTERM:
		log_info("Interrupt received. Stopping llmnrd.\n");
		iface_stop();
		llmnr_stop();
		break;
	case SIGHUP:
	default:
		/* ignore */
		break;
	}
}

static void register_signal(int sig, void (*handler)(int))
{
	sigset_t block_mask;
	struct sigaction saction;

	sigfillset(&block_mask);

	saction.sa_handler = handler;
	saction.sa_mask = block_mask;

	if (sigaction(sig, &saction, NULL) != 0) {
		log_err("Failed to register signal handler for %s (%d)\n",
			strsignal(sig), sig);
	}
}

int main(int argc, char **argv)
{
	int c, ret = EXIT_FAILURE;
	long num_arg;
	bool daemonize = false, ipv6 = false;
	char *hostname = NULL;
	uint16_t port = LLMNR_UDP_PORT;

	while ((c = getopt_long(argc, argv, short_opts, long_opts, NULL)) != -1) {
		switch (c) {
		case 'd':
			daemonize = true;
			break;
		case 'H':
			hostname = xstrdup(optarg);
			break;
		case 'p':
			num_arg = strtol(optarg, NULL, 0);
			if (num_arg < 0 || num_arg > UINT16_MAX) {
				log_err("Invalid port number: %ld\n", num_arg);
				return EXIT_FAILURE;
			}
			port = num_arg;
		case '6':
			ipv6 = true;
			break;
		case 'V':
			version_and_exit();
		case 'h':
			usage_and_exit(EXIT_SUCCESS);
		default:
			usage_and_exit(EXIT_FAILURE);
		}
	}

	register_signal(SIGINT, signal_handler);
	register_signal(SIGQUIT, signal_handler);
	register_signal(SIGTERM, signal_handler);
	register_signal(SIGHUP, signal_handler);

	if (!hostname) {
		/* TODO: Consider hostname changing at runtime */
		hostname = xmalloc(255);
		if (gethostname(hostname, 255) != 0) {
			log_err("Failed to get hostname");
			return EXIT_FAILURE;
		}
	}

	if (daemonize) {
		if (daemon(0, 0) != 0) {
			log_err("Failed to daemonize process: %s\n", strerror(errno));
			return EXIT_FAILURE;
		}
	}

	if (llmnr_init(hostname, port, ipv6) < 0)
		goto out;

	if (iface_start_thread() < 0)
		goto out;

	ret = llmnr_run();
out:
	free(hostname);
	return ret;
}
