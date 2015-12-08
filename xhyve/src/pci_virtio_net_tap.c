/*-
 * Copyright (c) 2011 NetApp, Inc.
 * Copyright (c) 2015 xhyve developers
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY NETAPP, INC ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL NETAPP, INC OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 *
 * $FreeBSD$
 */

#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <strings.h>
#include <pthread.h>
#include <fcntl.h>
#include <unistd.h>
#include <errno.h>
#include <assert.h>
#include <sys/select.h>
#include <sys/param.h>
#include <sys/uio.h>
#include <sys/ioctl.h>
#include <net/ethernet.h>
#include <xhyve/support/misc.h>
#include <xhyve/support/atomic.h>
#include <xhyve/support/linker_set.h>
#include <xhyve/support/md5.h>
#include <xhyve/xhyve.h>
#include <xhyve/pci_emul.h>
#include <xhyve/mevent.h>
#include <xhyve/virtio.h>

#define USE_MEVENT 0

#define VTNET_RINGSZ 1024
#define VTNET_MAXSEGS 32

/*
 * Host capabilities.  Note that we only offer a few of these.
 */
// #define VIRTIO_NET_F_CSUM (1 << 0) /* host handles partial cksum */
// #define VIRTIO_NET_F_GUEST_CSUM (1 << 1) /* guest handles partial cksum */
#define VIRTIO_NET_F_MAC (1 << 5) /* host supplies MAC */
// #define VIRTIO_NET_F_GSO_DEPREC (1 << 6) /* deprecated: host handles GSO */
// #define VIRTIO_NET_F_GUEST_TSO4 (1 << 7) /* guest can rcv TSOv4 */
// #define VIRTIO_NET_F_GUEST_TSO6 (1 << 8) /* guest can rcv TSOv6 */
// #define VIRTIO_NET_F_GUEST_ECN (1 << 9) /* guest can rcv TSO with ECN */
// #define VIRTIO_NET_F_GUEST_UFO (1 << 10) /* guest can rcv UFO */
// #define VIRTIO_NET_F_HOST_TSO4 (1 << 11) /* host can rcv TSOv4 */
// #define VIRTIO_NET_F_HOST_TSO6 (1 << 12) /* host can rcv TSOv6 */
// #define VIRTIO_NET_F_HOST_ECN (1 << 13) /* host can rcv TSO with ECN */
// #define VIRTIO_NET_F_HOST_UFO (1 << 14) /* host can rcv UFO */
#define VIRTIO_NET_F_MRG_RXBUF (1 << 15) /* host can merge RX buffers */
#define VIRTIO_NET_F_STATUS (1 << 16) /* config status field available */
// #define VIRTIO_NET_F_CTRL_VQ (1 << 17) /* control channel available */
// #define VIRTIO_NET_F_CTRL_RX (1 << 18) /* control channel RX mode support */
// #define VIRTIO_NET_F_CTRL_VLAN (1 << 19) /* control channel VLAN filtering */
// #define VIRTIO_NET_F_GUEST_ANNOUNCE (1 << 21) /* guest can send gratuit. pkts */

#define VTNET_S_HOSTCAPS \
	(VIRTIO_NET_F_MAC | VIRTIO_NET_F_MRG_RXBUF | VIRTIO_NET_F_STATUS | \
	VIRTIO_F_NOTIFY_ON_EMPTY)

#define ETHER_IS_MULTICAST(addr) (*(addr) & 0x01) /* is address mcast/bcast? */

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wpacked"

/*
 * PCI config-space "registers"
 */
struct virtio_net_config {
	uint8_t mac[6];
	uint16_t status;
} __packed;

/*
 * Queue definitions.
 */
#define VTNET_RXQ 0
#define VTNET_TXQ 1
// #define VTNET_CTLQ 2 /* NB: not yet supported */
#define VTNET_MAXQ 3

/*
 * Fixed network header size
 */
struct virtio_net_rxhdr {
	uint8_t vrh_flags;
	uint8_t vrh_gso_type;
	uint16_t vrh_hdr_len;
	uint16_t vrh_gso_size;
	uint16_t vrh_csum_start;
	uint16_t vrh_csum_offset;
	uint16_t vrh_bufs;
} __packed;

#pragma clang diagnostic pop

/*
 * Debug printf
 */
static int pci_vtnet_debug;
#define DPRINTF(params) if (pci_vtnet_debug) printf params
#define WPRINTF(params) printf params

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wpadded"
/*
 * Per-device softc
 */
struct pci_vtnet_softc {
	struct virtio_softc vsc_vs;
	struct vqueue_info vsc_queues[VTNET_MAXQ - 1];
	pthread_mutex_t vsc_mtx;
	struct mevent *vsc_mevp;
	int vsc_tapfd;
	int vsc_rx_ready;
	volatile int resetting;/* set and checked outside lock */
	uint64_t vsc_features; /* negotiated features */
	struct virtio_net_config vsc_config;
	pthread_mutex_t rx_mtx;
	int rx_in_progress;
	int rx_vhdrlen;
	int rx_merge; /* merged rx bufs in use */
	pthread_t tx_tid;
	pthread_mutex_t tx_mtx;
	pthread_cond_t tx_cond;
	int tx_in_progress;
};
#pragma clang diagnostic pop

static void pci_vtnet_reset(void *);
/* static void pci_vtnet_notify(void *, struct vqueue_info *); */
static int pci_vtnet_cfgread(void *, int, int, uint32_t *);
static int pci_vtnet_cfgwrite(void *, int, int, uint32_t);
static void pci_vtnet_neg_features(void *, uint64_t);

static struct virtio_consts vtnet_vi_consts = {
	"vtnet",		/* our name */
	VTNET_MAXQ - 1,		/* we currently support 2 virtqueues */
	sizeof(struct virtio_net_config), /* config reg size */
	pci_vtnet_reset,	/* reset */
	NULL,			/* device-wide qnotify -- not used */
	pci_vtnet_cfgread,	/* read PCI config */
	pci_vtnet_cfgwrite,	/* write PCI config */
	pci_vtnet_neg_features,	/* apply negotiated features */
	VTNET_S_HOSTCAPS,	/* our capabilities */
};

/*
 * If the transmit thread is active then stall until it is done.
 */
static void
pci_vtnet_txwait(struct pci_vtnet_softc *sc)
{

	pthread_mutex_lock(&sc->tx_mtx);
	while (sc->tx_in_progress) {
		pthread_mutex_unlock(&sc->tx_mtx);
		usleep(10000);
		pthread_mutex_lock(&sc->tx_mtx);
	}
	pthread_mutex_unlock(&sc->tx_mtx);
}

/*
 * If the receive thread is active then stall until it is done.
 */
static void
pci_vtnet_rxwait(struct pci_vtnet_softc *sc)
{

	pthread_mutex_lock(&sc->rx_mtx);
	while (sc->rx_in_progress) {
		pthread_mutex_unlock(&sc->rx_mtx);
		usleep(10000);
		pthread_mutex_lock(&sc->rx_mtx);
	}
	pthread_mutex_unlock(&sc->rx_mtx);
}

static void
pci_vtnet_reset(void *vsc)
{
	struct pci_vtnet_softc *sc = vsc;

	DPRINTF(("vtnet: device reset requested !\n"));

	sc->resetting = 1;

	/*
	 * Wait for the transmit and receive threads to finish their
	 * processing.
	 */
	pci_vtnet_txwait(sc);
	pci_vtnet_rxwait(sc);

	sc->vsc_rx_ready = 0;
	sc->rx_merge = 1;
	sc->rx_vhdrlen = sizeof(struct virtio_net_rxhdr);

	/* now reset rings, MSI-X vectors, and negotiated capabilities */
	vi_reset_dev(&sc->vsc_vs);

	sc->resetting = 0;
}

/*
 * Called to send a buffer chain out to the tap device
 */
static void
pci_vtnet_tap_tx(struct pci_vtnet_softc *sc, struct iovec *iov, int iovcnt,
		 int len)
{
	static char pad[60]; /* all zero bytes */

	if (sc->vsc_tapfd == -1)
		return;

	/*
	 * If the length is < 60, pad out to that and add the
	 * extra zero'd segment to the iov. It is guaranteed that
	 * there is always an extra iov available by the caller.
	 */
	if (len < 60) {
		iov[iovcnt].iov_base = pad;
		iov[iovcnt].iov_len = (size_t) (60 - len);
		iovcnt++;
	}
	(void) writev(sc->vsc_tapfd, iov, iovcnt);
}

/*
 *  Called when there is read activity on the tap file descriptor.
 * Each buffer posted by the guest is assumed to be able to contain
 * an entire ethernet frame + rx header.
 *  MP note: the dummybuf is only used for discarding frames, so there
 * is no need for it to be per-vtnet or locked.
 */
static uint8_t dummybuf[2048];

static __inline struct iovec *
rx_iov_trim(struct iovec *iov, int *niov, int tlen)
{
	struct iovec *riov;

	/* XXX short-cut: assume first segment is >= tlen */
	assert(iov[0].iov_len >= ((size_t) tlen));

	iov[0].iov_len -= ((size_t) tlen);
	if (iov[0].iov_len == 0) {
		assert(*niov > 1);
		*niov -= 1;
		riov = &iov[1];
	} else {
		iov[0].iov_base = (void *)((uintptr_t)iov[0].iov_base +
			((size_t) tlen));
		riov = &iov[0];
	}

	return (riov);
}

static void
pci_vtnet_tap_rx(struct pci_vtnet_softc *sc)
{
	struct iovec iov[VTNET_MAXSEGS], *riov;
	struct vqueue_info *vq;
	void *vrx;
	int len, n;
	uint16_t idx;

	/*
	 * Should never be called without a valid tap fd
	 */
	assert(sc->vsc_tapfd != -1);

	/*
	 * But, will be called when the rx ring hasn't yet
	 * been set up or the guest is resetting the device.
	 */
	if (!sc->vsc_rx_ready || sc->resetting) {
		/*
		 * Drop the packet and try later.
		 */
		(void) read(sc->vsc_tapfd, dummybuf, sizeof(dummybuf));
		return;
	}

	/*
	 * Check for available rx buffers
	 */
	vq = &sc->vsc_queues[VTNET_RXQ];
	if (!vq_has_descs(vq)) {
		/*
		 * Drop the packet and try later.  Interrupt on
		 * empty, if that's negotiated.
		 */
		(void) read(sc->vsc_tapfd, dummybuf, sizeof(dummybuf));
		vq_endchains(vq, 1);
		return;
	}

	do {
		/*
		 * Get descriptor chain.
		 */
		n = vq_getchain(vq, &idx, iov, VTNET_MAXSEGS, NULL);
		assert(n >= 1 && n <= VTNET_MAXSEGS);

		/*
		 * Get a pointer to the rx header, and use the
		 * data immediately following it for the packet buffer.
		 */
		vrx = iov[0].iov_base;
		riov = rx_iov_trim(iov, &n, sc->rx_vhdrlen);

		len = (int) readv(sc->vsc_tapfd, riov, n);

		if (len < 0 && errno == EWOULDBLOCK) {
			/*
			 * No more packets, but still some avail ring
			 * entries.  Interrupt if needed/appropriate.
			 */
			vq_retchain(vq);
			vq_endchains(vq, 0);
			return;
		}

		/*
		 * The only valid field in the rx packet header is the
		 * number of buffers if merged rx bufs were negotiated.
		 */
		memset(vrx, 0, sc->rx_vhdrlen);

		if (sc->rx_merge) {
			struct virtio_net_rxhdr *vrxh;

			vrxh = vrx;
			vrxh->vrh_bufs = 1;
		}

		/*
		 * Release this chain and handle more chains.
		 */
		vq_relchain(vq, idx, ((uint32_t) (len + sc->rx_vhdrlen)));
	} while (vq_has_descs(vq));

	/* Interrupt if needed, including for NOTIFY_ON_EMPTY. */
	vq_endchains(vq, 1);
}

#if USE_MEVENT
static void
pci_vtnet_tap_callback(UNUSED int fd, UNUSED enum ev_type type, void *param)
{
	struct pci_vtnet_softc *sc = param;

	pthread_mutex_lock(&sc->rx_mtx);
	sc->rx_in_progress = 1;
	pci_vtnet_tap_rx(sc);
	sc->rx_in_progress = 0;
	pthread_mutex_unlock(&sc->rx_mtx);

}

#else /* !USE_MEVENT */

static void *
pci_vtnet_tap_select_func(void *vsc) {
	struct pci_vtnet_softc *sc;
	fd_set rfd;

	sc = vsc;

	assert(sc);
	assert(sc->vsc_tapfd != -1);

	FD_ZERO(&rfd);
	FD_SET(sc->vsc_tapfd, &rfd);

	while (1) {
		if (select((sc->vsc_tapfd + 1), &rfd, NULL, NULL, NULL) == -1) {
			abort();
		}

		pthread_mutex_lock(&sc->rx_mtx);
		sc->rx_in_progress = 1;
		pci_vtnet_tap_rx(sc);
		sc->rx_in_progress = 0;
		pthread_mutex_unlock(&sc->rx_mtx);
	}

	return (NULL);
}
#endif

static void
pci_vtnet_ping_rxq(void *vsc, struct vqueue_info *vq)
{
	struct pci_vtnet_softc *sc = vsc;

	/*
	 * A qnotify means that the rx process can now begin
	 */
	if (sc->vsc_rx_ready == 0) {
		sc->vsc_rx_ready = 1;
		vq->vq_used->vu_flags |= VRING_USED_F_NO_NOTIFY;
	}
}

static void
pci_vtnet_proctx(struct pci_vtnet_softc *sc, struct vqueue_info *vq)
{
	struct iovec iov[VTNET_MAXSEGS + 1];
	int i, n;
	int plen, tlen;
	uint16_t idx;

	/*
	 * Obtain chain of descriptors.  The first one is
	 * really the header descriptor, so we need to sum
	 * up two lengths: packet length and transfer length.
	 */
	n = vq_getchain(vq, &idx, iov, VTNET_MAXSEGS, NULL);
	assert(n >= 1 && n <= VTNET_MAXSEGS);
	plen = 0;
	tlen = (int) iov[0].iov_len;
	for (i = 1; i < n; i++) {
		plen += iov[i].iov_len;
		tlen += iov[i].iov_len;
	}

	DPRINTF(("virtio: packet send, %d bytes, %d segs\n\r", plen, n));
	pci_vtnet_tap_tx(sc, &iov[1], n - 1, plen);

	/* chain is processed, release it and set tlen */
	vq_relchain(vq, idx, ((uint32_t) tlen));
}

static void
pci_vtnet_ping_txq(void *vsc, struct vqueue_info *vq)
{
	struct pci_vtnet_softc *sc = vsc;

	/*
	 * Any ring entries to process?
	 */
	if (!vq_has_descs(vq))
		return;

	/* Signal the tx thread for processing */
	pthread_mutex_lock(&sc->tx_mtx);
	vq->vq_used->vu_flags |= VRING_USED_F_NO_NOTIFY;
	if (sc->tx_in_progress == 0)
		pthread_cond_signal(&sc->tx_cond);
	pthread_mutex_unlock(&sc->tx_mtx);
}

/*
 * Thread which will handle processing of TX desc
 */
static void *
pci_vtnet_tx_thread(void *param)
{
	struct pci_vtnet_softc *sc = param;
	struct vqueue_info *vq;
	int error;

	vq = &sc->vsc_queues[VTNET_TXQ];

	/*
	 * Let us wait till the tx queue pointers get initialised &
	 * first tx signaled
	 */
	pthread_mutex_lock(&sc->tx_mtx);
	error = pthread_cond_wait(&sc->tx_cond, &sc->tx_mtx);
	assert(error == 0);

	for (;;) {
		/* note - tx mutex is locked here */
		while (sc->resetting || !vq_has_descs(vq)) {
			vq->vq_used->vu_flags &= ~VRING_USED_F_NO_NOTIFY;
			mb();
			if (!sc->resetting && vq_has_descs(vq))
				break;

			sc->tx_in_progress = 0;
			error = pthread_cond_wait(&sc->tx_cond, &sc->tx_mtx);
			assert(error == 0);
		}
		vq->vq_used->vu_flags |= VRING_USED_F_NO_NOTIFY;
		sc->tx_in_progress = 1;
		pthread_mutex_unlock(&sc->tx_mtx);

		do {
			/*
			 * Run through entries, placing them into
			 * iovecs and sending when an end-of-packet
			 * is found
			 */
			pci_vtnet_proctx(sc, vq);
		} while (vq_has_descs(vq));

		/*
		 * Generate an interrupt if needed.
		 */
		vq_endchains(vq, 1);

		pthread_mutex_lock(&sc->tx_mtx);
	}
}

#ifdef notyet
static void
pci_vtnet_ping_ctlq(void *vsc, struct vqueue_info *vq)
{

	DPRINTF(("vtnet: control qnotify!\n\r"));
}
#endif

static int
pci_vtnet_parsemac(char *mac_str, uint8_t *mac_addr)
{
        struct ether_addr *ea;
        char *tmpstr;
        char zero_addr[ETHER_ADDR_LEN] = { 0, 0, 0, 0, 0, 0 };

        tmpstr = strsep(&mac_str,"=");
       
        if ((mac_str != NULL) && (!strcmp(tmpstr,"mac"))) {
                ea = ether_aton(mac_str);

                if (ea == NULL || ETHER_IS_MULTICAST(ea->octet) ||
                    memcmp(ea->octet, zero_addr, ETHER_ADDR_LEN) == 0) {
			fprintf(stderr, "Invalid MAC %s\n", mac_str);
                        return (EINVAL);
                } else
                        memcpy(mac_addr, ea->octet, ETHER_ADDR_LEN);
        }

        return (0);
}


static int
pci_vtnet_init(struct pci_devinst *pi, char *opts)
{
	MD5_CTX mdctx;
	unsigned char digest[16];
	char nstr[80];
	struct pci_vtnet_softc *sc;
	char *devname;
	char *vtopts;
	int mac_provided;
#if !USE_MEVENT
	pthread_t sthrd;
#endif

	sc = calloc(1, sizeof(struct pci_vtnet_softc));

	pthread_mutex_init(&sc->vsc_mtx, NULL);

	vi_softc_linkup(&sc->vsc_vs, &vtnet_vi_consts, sc, pi, sc->vsc_queues);
	sc->vsc_vs.vs_mtx = &sc->vsc_mtx;

	sc->vsc_queues[VTNET_RXQ].vq_qsize = VTNET_RINGSZ;
	sc->vsc_queues[VTNET_RXQ].vq_notify = pci_vtnet_ping_rxq;
	sc->vsc_queues[VTNET_TXQ].vq_qsize = VTNET_RINGSZ;
	sc->vsc_queues[VTNET_TXQ].vq_notify = pci_vtnet_ping_txq;
#ifdef notyet
	sc->vsc_queues[VTNET_CTLQ].vq_qsize = VTNET_RINGSZ;
        sc->vsc_queues[VTNET_CTLQ].vq_notify = pci_vtnet_ping_ctlq;
#endif
 
	/*
	 * Attempt to open the tap device and read the MAC address
	 * if specified
	 */
	mac_provided = 0;
	sc->vsc_tapfd = -1;
	if (opts != NULL) {
		char tbuf[80];
		int err;

		devname = vtopts = strdup(opts);
		(void) strsep(&vtopts, ",");

		if (vtopts != NULL) {
			err = pci_vtnet_parsemac(vtopts, sc->vsc_config.mac);
			if (err != 0) {
				free(devname);
				return (err);
			}
			mac_provided = 1;
		}

		strcpy(tbuf, "/dev/");
		strlcat(tbuf, devname, sizeof(tbuf));

		free(devname);

		sc->vsc_tapfd = open(tbuf, O_RDWR);
		if (sc->vsc_tapfd == -1) {
			WPRINTF(("open of tap device %s failed\n", tbuf));
		} else {
			/*
			 * Set non-blocking and register for read
			 * notifications with the event loop
			 */
			int opt = 1;
			if (ioctl(sc->vsc_tapfd, FIONBIO, &opt) < 0) {
				WPRINTF(("tap device O_NONBLOCK failed\n"));
				close(sc->vsc_tapfd);
				sc->vsc_tapfd = -1;
			}

#if USE_MEVENT
			sc->vsc_mevp = mevent_add(sc->vsc_tapfd,
						  EVF_READ,
						  pci_vtnet_tap_callback,
						  sc);
			if (sc->vsc_mevp == NULL) {
				WPRINTF(("Could not register event\n"));
				close(sc->vsc_tapfd);
				sc->vsc_tapfd = -1;
			}

#else /* !USE_MEVENT */
			if (pthread_create(&sthrd, NULL, pci_vtnet_tap_select_func, sc)) {
				WPRINTF(("Could not create tap receive thread\n"));
				close(sc->vsc_tapfd);
				sc->vsc_tapfd = -1;
			}
#endif
		}
	}

	/*
	 * The default MAC address is the standard NetApp OUI of 00-a0-98,
	 * followed by an MD5 of the PCI slot/func number and dev name
	 */
	if (!mac_provided) {
		snprintf(nstr, sizeof(nstr), "%d-%d-%s", pi->pi_slot,
		    pi->pi_func, vmname);

		MD5Init(&mdctx);
		MD5Update(&mdctx, nstr, ((unsigned int) strlen(nstr)));
		MD5Final(digest, &mdctx);

		sc->vsc_config.mac[0] = 0x00;
		sc->vsc_config.mac[1] = 0xa0;
		sc->vsc_config.mac[2] = 0x98;
		sc->vsc_config.mac[3] = digest[0];
		sc->vsc_config.mac[4] = digest[1];
		sc->vsc_config.mac[5] = digest[2];
	}

	/* initialize config space */
	pci_set_cfgdata16(pi, PCIR_DEVICE, VIRTIO_DEV_NET);
	pci_set_cfgdata16(pi, PCIR_VENDOR, VIRTIO_VENDOR);
	pci_set_cfgdata8(pi, PCIR_CLASS, PCIC_NETWORK);
	pci_set_cfgdata16(pi, PCIR_SUBDEV_0, VIRTIO_TYPE_NET);
	pci_set_cfgdata16(pi, PCIR_SUBVEND_0, VIRTIO_VENDOR);

	/* Link is up if we managed to open tap device. */
	sc->vsc_config.status = (opts == NULL || sc->vsc_tapfd >= 0);
	
	/* use BAR 1 to map MSI-X table and PBA, if we're using MSI-X */
	if (vi_intr_init(&sc->vsc_vs, 1, fbsdrun_virtio_msix()))
		return (1);

	/* use BAR 0 to map config regs in IO space */
	vi_set_io_bar(&sc->vsc_vs, 0);

	sc->resetting = 0;

	sc->rx_merge = 1;
	sc->rx_vhdrlen = sizeof(struct virtio_net_rxhdr);
	sc->rx_in_progress = 0;
	pthread_mutex_init(&sc->rx_mtx, NULL); 

	/* 
	 * Initialize tx semaphore & spawn TX processing thread.
	 * As of now, only one thread for TX desc processing is
	 * spawned. 
	 */
	sc->tx_in_progress = 0;
	pthread_mutex_init(&sc->tx_mtx, NULL);
	pthread_cond_init(&sc->tx_cond, NULL);
	pthread_create(&sc->tx_tid, NULL, pci_vtnet_tx_thread, (void *)sc);
	return (0);
}

static int
pci_vtnet_cfgwrite(void *vsc, int offset, int size, uint32_t value)
{
	struct pci_vtnet_softc *sc = vsc;
	void *ptr;

	if (offset < 6) {
		assert(offset + size <= 6);
		/*
		 * The driver is allowed to change the MAC address
		 */
		ptr = &sc->vsc_config.mac[offset];
		memcpy(ptr, &value, size);
	} else {
		/* silently ignore other writes */
		DPRINTF(("vtnet: write to readonly reg %d\n\r", offset));
	}

	return (0);
}

static int
pci_vtnet_cfgread(void *vsc, int offset, int size, uint32_t *retval)
{
	struct pci_vtnet_softc *sc = vsc;
	void *ptr;

	ptr = (uint8_t *)&sc->vsc_config + offset;
	memcpy(retval, ptr, size);
	return (0);
}

static void
pci_vtnet_neg_features(void *vsc, uint64_t negotiated_features)
{
	struct pci_vtnet_softc *sc = vsc;

	sc->vsc_features = negotiated_features;

	if (!(sc->vsc_features & VIRTIO_NET_F_MRG_RXBUF)) {
		sc->rx_merge = 0;
		/* non-merge rx header is 2 bytes shorter */
		sc->rx_vhdrlen -= 2;
	}
}

static struct pci_devemu pci_de_vnet_tap = {
	.pe_emu = 	"virtio-tap",
	.pe_init =	pci_vtnet_init,
	.pe_barwrite =	vi_pci_write,
	.pe_barread =	vi_pci_read
};
PCI_EMUL_SET(pci_de_vnet_tap);
