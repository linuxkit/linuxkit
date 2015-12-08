/*-
 * Copyright (c) 2000 - 2008 Søren Schmidt <sos@FreeBSD.org>
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer,
 *    without modification, immediately at the beginning of the file.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR ``AS IS'' AND ANY EXPRESS OR
 * IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES
 * OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED.
 * IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY DIRECT, INDIRECT,
 * INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT
 * NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
 * THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 * $FreeBSD$
 */

#pragma once

#include <xhyve/support/misc.h>

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wpadded"
#pragma clang diagnostic ignored "-Wpacked"

/* ATA/ATAPI device parameters */
struct ata_params {
/*000*/ u_int16_t       config;         /* configuration info */
#define ATA_PROTO_MASK                  0x8003
#define ATA_PROTO_ATAPI                 0x8000
#define ATA_PROTO_ATAPI_12              0x8000
#define ATA_PROTO_ATAPI_16              0x8001
#define ATA_PROTO_CFA                   0x848a
#define ATA_ATAPI_TYPE_MASK             0x1f00
#define ATA_ATAPI_TYPE_DIRECT           0x0000  /* disk/floppy */
#define ATA_ATAPI_TYPE_TAPE             0x0100  /* streaming tape */
#define ATA_ATAPI_TYPE_CDROM            0x0500  /* CD-ROM device */
#define ATA_ATAPI_TYPE_OPTICAL          0x0700  /* optical disk */
#define ATA_DRQ_MASK                    0x0060
#define ATA_DRQ_SLOW                    0x0000  /* cpu 3 ms delay */
#define ATA_DRQ_INTR                    0x0020  /* interrupt 10 ms delay */
#define ATA_DRQ_FAST                    0x0040  /* accel 50 us delay */
#define ATA_RESP_INCOMPLETE             0x0004

/*001*/ u_int16_t       cylinders;              /* # of cylinders */
/*002*/ u_int16_t       specconf;		/* specific configuration */
/*003*/ u_int16_t       heads;                  /* # heads */
	u_int16_t       obsolete4;
	u_int16_t       obsolete5;
/*006*/ u_int16_t       sectors;                /* # sectors/track */
/*007*/ u_int16_t       vendor7[3];
/*010*/ u_int8_t        serial[20];             /* serial number */
/*020*/ u_int16_t       retired20;
	u_int16_t       retired21;
	u_int16_t       obsolete22;
/*023*/ u_int8_t        revision[8];            /* firmware revision */
/*027*/ u_int8_t        model[40];              /* model name */
/*047*/ u_int16_t       sectors_intr;           /* sectors per interrupt */
/*048*/ u_int16_t       usedmovsd;              /* double word read/write? */
/*049*/ u_int16_t       capabilities1;
#define ATA_SUPPORT_DMA                 0x0100
#define ATA_SUPPORT_LBA                 0x0200
#define ATA_SUPPORT_IORDY               0x0400
#define ATA_SUPPORT_IORDYDIS            0x0800
#define ATA_SUPPORT_OVERLAP             0x4000

/*050*/ u_int16_t       capabilities2;
/*051*/ u_int16_t       retired_piomode;        /* PIO modes 0-2 */
#define ATA_RETIRED_PIO_MASK            0x0300

/*052*/ u_int16_t       retired_dmamode;        /* DMA modes */
#define ATA_RETIRED_DMA_MASK            0x0003

/*053*/ u_int16_t       atavalid;               /* fields valid */
#define ATA_FLAG_54_58                  0x0001  /* words 54-58 valid */
#define ATA_FLAG_64_70                  0x0002  /* words 64-70 valid */
#define ATA_FLAG_88                     0x0004  /* word 88 valid */

/*054*/ u_int16_t       current_cylinders;
/*055*/ u_int16_t       current_heads;
/*056*/ u_int16_t       current_sectors;
/*057*/ u_int16_t       current_size_1;
/*058*/ u_int16_t       current_size_2;
/*059*/ u_int16_t       multi;
#define ATA_MULTI_VALID                 0x0100

/*060*/ u_int16_t       lba_size_1;
	u_int16_t       lba_size_2;
	u_int16_t       obsolete62;
/*063*/ u_int16_t       mwdmamodes;             /* multiword DMA modes */
/*064*/ u_int16_t       apiomodes;              /* advanced PIO modes */

/*065*/ u_int16_t       mwdmamin;               /* min. M/W DMA time/word ns */
/*066*/ u_int16_t       mwdmarec;               /* rec. M/W DMA time ns */
/*067*/ u_int16_t       pioblind;               /* min. PIO cycle w/o flow */
/*068*/ u_int16_t       pioiordy;               /* min. PIO cycle IORDY flow */
/*069*/ u_int16_t       support3;
#define ATA_SUPPORT_RZAT                0x0020
#define ATA_SUPPORT_DRAT                0x4000
	u_int16_t       reserved70;
/*071*/ u_int16_t       rlsovlap;               /* rel time (us) for overlap */
/*072*/ u_int16_t       rlsservice;             /* rel time (us) for service */
	u_int16_t       reserved73;
	u_int16_t       reserved74;
/*075*/ u_int16_t       queue;
#define ATA_QUEUE_LEN(x)                ((x) & 0x001f)

/*76*/  u_int16_t       satacapabilities;
#define ATA_SATA_GEN1                   0x0002
#define ATA_SATA_GEN2                   0x0004
#define ATA_SATA_GEN3                   0x0008
#define ATA_SUPPORT_NCQ                 0x0100
#define ATA_SUPPORT_IFPWRMNGTRCV        0x0200
#define ATA_SUPPORT_PHYEVENTCNT         0x0400
#define ATA_SUPPORT_NCQ_UNLOAD          0x0800
#define ATA_SUPPORT_NCQ_PRIO            0x1000
#define ATA_SUPPORT_HAPST               0x2000
#define ATA_SUPPORT_DAPST               0x4000
#define ATA_SUPPORT_READLOGDMAEXT       0x8000

/*77*/  u_int16_t       satacapabilities2;
#define ATA_SATA_CURR_GEN_MASK          0x0006
#define ATA_SUPPORT_NCQ_STREAM          0x0010
#define ATA_SUPPORT_NCQ_QMANAGEMENT     0x0020
#define ATA_SUPPORT_RCVSND_FPDMA_QUEUED 0x0040
/*78*/  u_int16_t       satasupport;
#define ATA_SUPPORT_NONZERO             0x0002
#define ATA_SUPPORT_AUTOACTIVATE        0x0004
#define ATA_SUPPORT_IFPWRMNGT           0x0008
#define ATA_SUPPORT_INORDERDATA         0x0010
#define ATA_SUPPORT_ASYNCNOTIF          0x0020
#define ATA_SUPPORT_SOFTSETPRESERVE     0x0040
/*79*/  u_int16_t       sataenabled;
#define ATA_ENABLED_DAPST               0x0080

/*080*/ u_int16_t       version_major;
/*081*/ u_int16_t       version_minor;

	struct {
/*082/085*/ u_int16_t   command1;
#define ATA_SUPPORT_SMART               0x0001
#define ATA_SUPPORT_SECURITY            0x0002
#define ATA_SUPPORT_REMOVABLE           0x0004
#define ATA_SUPPORT_POWERMGT            0x0008
#define ATA_SUPPORT_PACKET              0x0010
#define ATA_SUPPORT_WRITECACHE          0x0020
#define ATA_SUPPORT_LOOKAHEAD           0x0040
#define ATA_SUPPORT_RELEASEIRQ          0x0080
#define ATA_SUPPORT_SERVICEIRQ          0x0100
#define ATA_SUPPORT_RESET               0x0200
#define ATA_SUPPORT_PROTECTED           0x0400
#define ATA_SUPPORT_WRITEBUFFER         0x1000
#define ATA_SUPPORT_READBUFFER          0x2000
#define ATA_SUPPORT_NOP                 0x4000

/*083/086*/ u_int16_t   command2;
#define ATA_SUPPORT_MICROCODE           0x0001
#define ATA_SUPPORT_QUEUED              0x0002
#define ATA_SUPPORT_CFA                 0x0004
#define ATA_SUPPORT_APM                 0x0008
#define ATA_SUPPORT_NOTIFY              0x0010
#define ATA_SUPPORT_STANDBY             0x0020
#define ATA_SUPPORT_SPINUP              0x0040
#define ATA_SUPPORT_MAXSECURITY         0x0100
#define ATA_SUPPORT_AUTOACOUSTIC        0x0200
#define ATA_SUPPORT_ADDRESS48           0x0400
#define ATA_SUPPORT_OVERLAY             0x0800
#define ATA_SUPPORT_FLUSHCACHE          0x1000
#define ATA_SUPPORT_FLUSHCACHE48        0x2000

/*084/087*/ u_int16_t   extension;
#define ATA_SUPPORT_SMARTLOG		0x0001
#define ATA_SUPPORT_SMARTTEST		0x0002
#define ATA_SUPPORT_MEDIASN		0x0004
#define ATA_SUPPORT_MEDIAPASS		0x0008
#define ATA_SUPPORT_STREAMING		0x0010
#define ATA_SUPPORT_GENLOG		0x0020
#define ATA_SUPPORT_WRITEDMAFUAEXT	0x0040
#define ATA_SUPPORT_WRITEDMAQFUAEXT	0x0080
#define ATA_SUPPORT_64BITWWN		0x0100
#define ATA_SUPPORT_UNLOAD		0x2000
	} __packed support, enabled;

/*088*/ u_int16_t       udmamodes;              /* UltraDMA modes */
/*089*/ u_int16_t       erase_time;             /* time req'd in 2min units */
/*090*/ u_int16_t       enhanced_erase_time;    /* time req'd in 2min units */
/*091*/ u_int16_t       apm_value;
/*092*/ u_int16_t       master_passwd_revision; /* password revision code */
/*093*/ u_int16_t       hwres;
#define ATA_CABLE_ID                    0x2000

/*094*/ u_int16_t       acoustic;
#define ATA_ACOUSTIC_CURRENT(x)         ((x) & 0x00ff)
#define ATA_ACOUSTIC_VENDOR(x)          (((x) & 0xff00) >> 8)

/*095*/ u_int16_t       stream_min_req_size;
/*096*/ u_int16_t       stream_transfer_time;
/*097*/ u_int16_t       stream_access_latency;
/*098*/ u_int32_t       stream_granularity;
/*100*/ u_int16_t       lba_size48_1;
	u_int16_t       lba_size48_2;
	u_int16_t       lba_size48_3;
	u_int16_t       lba_size48_4;
	u_int16_t       reserved104;
/*105*/	u_int16_t       max_dsm_blocks;
/*106*/	u_int16_t       pss;
#define ATA_PSS_LSPPS			0x000F
#define ATA_PSS_LSSABOVE512		0x1000
#define ATA_PSS_MULTLS			0x2000
#define ATA_PSS_VALID_MASK		0xC000
#define ATA_PSS_VALID_VALUE		0x4000
/*107*/ u_int16_t       isd;
/*108*/ u_int16_t       wwn[4];
	u_int16_t       reserved112[5];
/*117*/ u_int16_t       lss_1;
/*118*/ u_int16_t       lss_2;
/*119*/ u_int16_t       support2;
#define ATA_SUPPORT_WRITEREADVERIFY	0x0002
#define ATA_SUPPORT_WRITEUNCORREXT	0x0004
#define ATA_SUPPORT_RWLOGDMAEXT		0x0008
#define ATA_SUPPORT_MICROCODE3		0x0010
#define ATA_SUPPORT_FREEFALL		0x0020
/*120*/ u_int16_t       enabled2;
	u_int16_t       reserved121[6];
/*127*/ u_int16_t       removable_status;
/*128*/ u_int16_t       security_status;
#define ATA_SECURITY_LEVEL		0x0100	/* 0: high, 1: maximum */
#define ATA_SECURITY_ENH_SUPP		0x0020	/* enhanced erase supported */
#define ATA_SECURITY_COUNT_EXP		0x0010	/* count expired */
#define ATA_SECURITY_FROZEN		0x0008	/* security config is frozen */
#define ATA_SECURITY_LOCKED		0x0004	/* drive is locked */
#define ATA_SECURITY_ENABLED		0x0002	/* ATA Security is enabled */
#define ATA_SECURITY_SUPPORTED		0x0001	/* ATA Security is supported */

	u_int16_t       reserved129[31];
/*160*/ u_int16_t       cfa_powermode1;
	u_int16_t       reserved161;
/*162*/ u_int16_t       cfa_kms_support;
/*163*/ u_int16_t       cfa_trueide_modes;
/*164*/ u_int16_t       cfa_memory_modes;
	u_int16_t       reserved165[4];
/*169*/	u_int16_t       support_dsm;
#define ATA_SUPPORT_DSM_TRIM		0x0001
	u_int16_t       reserved170[6];
/*176*/ u_int8_t        media_serial[60];
/*206*/ u_int16_t       sct;
	u_int16_t       reserved206[2];
/*209*/ u_int16_t       lsalign;
/*210*/ u_int16_t       wrv_sectors_m3_1;
	u_int16_t       wrv_sectors_m3_2;
/*212*/ u_int16_t       wrv_sectors_m2_1;
	u_int16_t       wrv_sectors_m2_2;
/*214*/ u_int16_t       nv_cache_caps;
/*215*/ u_int16_t       nv_cache_size_1;
	u_int16_t       nv_cache_size_2;
/*217*/ u_int16_t       media_rotation_rate;
#define ATA_RATE_NOT_REPORTED		0x0000
#define ATA_RATE_NON_ROTATING		0x0001
	u_int16_t       reserved218;
/*219*/ u_int16_t       nv_cache_opt;
/*220*/ u_int16_t       wrv_mode;
	u_int16_t       reserved221;
/*222*/ u_int16_t       transport_major;
/*223*/ u_int16_t       transport_minor;
	u_int16_t       reserved224[31];
/*255*/ u_int16_t       integrity;
} __packed;

/* ATA Dataset Management */
#define ATA_DSM_BLK_SIZE	512
#define ATA_DSM_BLK_RANGES	64
#define ATA_DSM_RANGE_SIZE	8
#define ATA_DSM_RANGE_MAX	65535

/*
 * ATA Device Register
 *
 * bit 7 Obsolete (was 1 in early ATA specs)
 * bit 6 Sets LBA/CHS mode. 1=LBA, 0=CHS 
 * bit 5 Obsolete (was 1 in early ATA specs)
 * bit 4 1 = Slave Drive, 0 = Master Drive
 * bit 3-0 In LBA mode, 27-24 of address. In CHS mode, head number
*/

#define ATA_DEV_MASTER		0x00
#define ATA_DEV_SLAVE		0x10
#define ATA_DEV_LBA		0x40

/* ATA limits */
#define ATA_MAX_28BIT_LBA	268435455UL

/* ATA Status Register */
#define ATA_STATUS_ERROR	0x01
#define ATA_STATUS_DEVICE_FAULT	0x20

/* ATA Error Register */
#define ATA_ERROR_ABORT		0x04
#define ATA_ERROR_ID_NOT_FOUND	0x10

/* ATA HPA Features */
#define ATA_HPA_FEAT_MAX_ADDR	0x00
#define ATA_HPA_FEAT_SET_PWD	0x01
#define ATA_HPA_FEAT_LOCK	0x02
#define ATA_HPA_FEAT_UNLOCK	0x03
#define ATA_HPA_FEAT_FREEZE	0x04

/* ATA transfer modes */
#define ATA_MODE_MASK           0x0f
#define ATA_DMA_MASK            0xf0
#define ATA_PIO                 0x00
#define ATA_PIO0                0x08
#define ATA_PIO1                0x09
#define ATA_PIO2                0x0a
#define ATA_PIO3                0x0b
#define ATA_PIO4                0x0c
#define ATA_PIO_MAX             0x0f
#define ATA_DMA                 0x10
#define ATA_WDMA0               0x20
#define ATA_WDMA1               0x21
#define ATA_WDMA2               0x22
#define ATA_UDMA0               0x40
#define ATA_UDMA1               0x41
#define ATA_UDMA2               0x42
#define ATA_UDMA3               0x43
#define ATA_UDMA4               0x44
#define ATA_UDMA5               0x45
#define ATA_UDMA6               0x46
#define ATA_SA150               0x47
#define ATA_SA300               0x48
#define ATA_DMA_MAX             0x4f


/* ATA commands */
#define ATA_NOP                         0x00    /* NOP */
#define         ATA_NF_FLUSHQUEUE       0x00    /* flush queued cmd's */
#define         ATA_NF_AUTOPOLL         0x01    /* start autopoll function */
#define ATA_DATA_SET_MANAGEMENT		0x06
#define 	ATA_DSM_TRIM		0x01
#define ATA_DEVICE_RESET                0x08    /* reset device */
#define ATA_READ                        0x20    /* read */
#define ATA_READ48                      0x24    /* read 48bit LBA */
#define ATA_READ_DMA48                  0x25    /* read DMA 48bit LBA */
#define ATA_READ_DMA_QUEUED48           0x26    /* read DMA QUEUED 48bit LBA */
#define ATA_READ_NATIVE_MAX_ADDRESS48   0x27    /* read native max addr 48bit */
#define ATA_READ_MUL48                  0x29    /* read multi 48bit LBA */
#define ATA_READ_STREAM_DMA48           0x2a    /* read DMA stream 48bit LBA */
#define ATA_READ_LOG_EXT                0x2f    /* read log ext - PIO Data-In */
#define ATA_READ_STREAM48               0x2b    /* read stream 48bit LBA */
#define ATA_WRITE                       0x30    /* write */
#define ATA_WRITE48                     0x34    /* write 48bit LBA */
#define ATA_WRITE_DMA48                 0x35    /* write DMA 48bit LBA */
#define ATA_WRITE_DMA_QUEUED48          0x36    /* write DMA QUEUED 48bit LBA*/
#define ATA_SET_MAX_ADDRESS48           0x37    /* set max address 48bit */
#define ATA_WRITE_MUL48                 0x39    /* write multi 48bit LBA */
#define ATA_WRITE_STREAM_DMA48          0x3a
#define ATA_WRITE_STREAM48              0x3b
#define ATA_WRITE_DMA_FUA48             0x3d
#define ATA_WRITE_DMA_QUEUED_FUA48      0x3e
#define ATA_WRITE_LOG_EXT               0x3f
#define ATA_READ_VERIFY                 0x40
#define ATA_READ_VERIFY48               0x42
#define ATA_READ_LOG_DMA_EXT            0x47    /* read log DMA ext - PIO Data-In */
#define ATA_READ_FPDMA_QUEUED           0x60    /* read DMA NCQ */
#define ATA_WRITE_FPDMA_QUEUED          0x61    /* write DMA NCQ */
#define ATA_NCQ_NON_DATA		0x63	/* NCQ non-data command */
#define ATA_SEND_FPDMA_QUEUED           0x64    /* send DMA NCQ */
#define		ATA_SFPDMA_DSM		0x00	/* Data set management */
#define			ATA_SFPDMA_DSM_TRIM	0x01	/* Set trim bit in auxilary */
#define		ATA_SFPDMA_HYBRID_EVICT	0x01	/* Hybrid Evict */
#define		ATA_SFPDMA_WLDMA	0x02	/* Write Log DMA EXT */
#define ATA_RECV_FPDMA_QUEUED           0x65    /* recieve DMA NCQ */
#define ATA_SEP_ATTN                    0x67    /* SEP request */
#define ATA_SEEK                        0x70    /* seek */
#define ATA_PACKET_CMD                  0xa0    /* packet command */
#define ATA_ATAPI_IDENTIFY              0xa1    /* get ATAPI params*/
#define ATA_SERVICE                     0xa2    /* service command */
#define ATA_SMART_CMD                   0xb0    /* SMART command */
#define ATA_CFA_ERASE                   0xc0    /* CFA erase */
#define ATA_READ_MUL                    0xc4    /* read multi */
#define ATA_WRITE_MUL                   0xc5    /* write multi */
#define ATA_SET_MULTI                   0xc6    /* set multi size */
#define ATA_READ_DMA_QUEUED             0xc7    /* read DMA QUEUED */
#define ATA_READ_DMA                    0xc8    /* read DMA */
#define ATA_WRITE_DMA                   0xca    /* write DMA */
#define ATA_WRITE_DMA_QUEUED            0xcc    /* write DMA QUEUED */
#define ATA_WRITE_MUL_FUA48             0xce
#define ATA_STANDBY_IMMEDIATE           0xe0    /* standby immediate */
#define ATA_IDLE_IMMEDIATE              0xe1    /* idle immediate */
#define ATA_STANDBY_CMD                 0xe2    /* standby */
#define ATA_IDLE_CMD                    0xe3    /* idle */
#define ATA_READ_BUFFER                 0xe4    /* read buffer */
#define ATA_READ_PM                     0xe4    /* read portmultiplier */
#define ATA_SLEEP                       0xe6    /* sleep */
#define ATA_FLUSHCACHE                  0xe7    /* flush cache to disk */
#define ATA_WRITE_PM                    0xe8    /* write portmultiplier */
#define ATA_FLUSHCACHE48                0xea    /* flush cache to disk */
#define ATA_ATA_IDENTIFY                0xec    /* get ATA params */
#define ATA_SETFEATURES                 0xef    /* features command */
#define         ATA_SF_SETXFER          0x03    /* set transfer mode */
#define         ATA_SF_ENAB_WCACHE      0x02    /* enable write cache */
#define         ATA_SF_DIS_WCACHE       0x82    /* disable write cache */
#define         ATA_SF_ENAB_PUIS        0x06    /* enable PUIS */
#define         ATA_SF_DIS_PUIS         0x86    /* disable PUIS */
#define         ATA_SF_PUIS_SPINUP      0x07    /* PUIS spin-up */
#define         ATA_SF_ENAB_RCACHE      0xaa    /* enable readahead cache */
#define         ATA_SF_DIS_RCACHE       0x55    /* disable readahead cache */
#define         ATA_SF_ENAB_RELIRQ      0x5d    /* enable release interrupt */
#define         ATA_SF_DIS_RELIRQ       0xdd    /* disable release interrupt */
#define         ATA_SF_ENAB_SRVIRQ      0x5e    /* enable service interrupt */
#define         ATA_SF_DIS_SRVIRQ       0xde    /* disable service interrupt */
#define ATA_SECURITY_SET_PASSWORD       0xf1    /* set drive password */
#define ATA_SECURITY_UNLOCK             0xf2    /* unlock drive using passwd */
#define ATA_SECURITY_ERASE_PREPARE      0xf3    /* prepare to erase drive */
#define ATA_SECURITY_ERASE_UNIT         0xf4    /* erase all blocks on drive */
#define ATA_SECURITY_FREEZE_LOCK        0xf5    /* freeze security config */
#define ATA_SECURITY_DISABLE_PASSWORD   0xf6    /* disable drive password */
#define ATA_READ_NATIVE_MAX_ADDRESS     0xf8    /* read native max address */
#define ATA_SET_MAX_ADDRESS             0xf9    /* set max address */


/* ATAPI commands */
#define ATAPI_TEST_UNIT_READY           0x00    /* check if device is ready */
#define ATAPI_REZERO                    0x01    /* rewind */
#define ATAPI_REQUEST_SENSE             0x03    /* get sense data */
#define ATAPI_FORMAT                    0x04    /* format unit */
#define ATAPI_READ                      0x08    /* read data */
#define ATAPI_WRITE                     0x0a    /* write data */
#define ATAPI_WEOF                      0x10    /* write filemark */
#define         ATAPI_WF_WRITE          0x01
#define ATAPI_SPACE                     0x11    /* space command */
#define         ATAPI_SP_FM             0x01
#define         ATAPI_SP_EOD            0x03
#define ATAPI_INQUIRY			0x12	/* get inquiry data */
#define ATAPI_MODE_SELECT               0x15    /* mode select */
#define ATAPI_ERASE                     0x19    /* erase */
#define ATAPI_MODE_SENSE                0x1a    /* mode sense */
#define ATAPI_START_STOP                0x1b    /* start/stop unit */
#define         ATAPI_SS_LOAD           0x01
#define         ATAPI_SS_RETENSION      0x02
#define         ATAPI_SS_EJECT          0x04
#define ATAPI_PREVENT_ALLOW             0x1e    /* media removal */
#define ATAPI_READ_FORMAT_CAPACITIES    0x23    /* get format capacities */
#define ATAPI_READ_CAPACITY             0x25    /* get volume capacity */
#define ATAPI_READ_BIG                  0x28    /* read data */
#define ATAPI_WRITE_BIG                 0x2a    /* write data */
#define ATAPI_LOCATE                    0x2b    /* locate to position */
#define ATAPI_READ_POSITION             0x34    /* read position */
#define ATAPI_SYNCHRONIZE_CACHE         0x35    /* flush buf, close channel */
#define ATAPI_WRITE_BUFFER              0x3b    /* write device buffer */
#define ATAPI_READ_BUFFER               0x3c    /* read device buffer */
#define ATAPI_READ_SUBCHANNEL           0x42    /* get subchannel info */
#define ATAPI_READ_TOC                  0x43    /* get table of contents */
#define ATAPI_PLAY_10                   0x45    /* play by lba */
#define ATAPI_PLAY_MSF                  0x47    /* play by MSF address */
#define ATAPI_PLAY_TRACK                0x48    /* play by track number */
#define ATAPI_PAUSE                     0x4b    /* pause audio operation */
#define ATAPI_READ_DISK_INFO            0x51    /* get disk info structure */
#define ATAPI_READ_TRACK_INFO           0x52    /* get track info structure */
#define ATAPI_RESERVE_TRACK             0x53    /* reserve track */
#define ATAPI_SEND_OPC_INFO             0x54    /* send OPC structurek */
#define ATAPI_MODE_SELECT_BIG           0x55    /* set device parameters */
#define ATAPI_REPAIR_TRACK              0x58    /* repair track */
#define ATAPI_READ_MASTER_CUE           0x59    /* read master CUE info */
#define ATAPI_MODE_SENSE_BIG            0x5a    /* get device parameters */
#define ATAPI_CLOSE_TRACK               0x5b    /* close track/session */
#define ATAPI_READ_BUFFER_CAPACITY      0x5c    /* get buffer capicity */
#define ATAPI_SEND_CUE_SHEET            0x5d    /* send CUE sheet */
#define ATAPI_SERVICE_ACTION_IN         0x96	/* get service data */
#define ATAPI_BLANK                     0xa1    /* blank the media */
#define ATAPI_SEND_KEY                  0xa3    /* send DVD key structure */
#define ATAPI_REPORT_KEY                0xa4    /* get DVD key structure */
#define ATAPI_PLAY_12                   0xa5    /* play by lba */
#define ATAPI_LOAD_UNLOAD               0xa6    /* changer control command */
#define ATAPI_READ_STRUCTURE            0xad    /* get DVD structure */
#define ATAPI_PLAY_CD                   0xb4    /* universal play command */
#define ATAPI_SET_SPEED                 0xbb    /* set drive speed */
#define ATAPI_MECH_STATUS               0xbd    /* get changer status */
#define ATAPI_READ_CD                   0xbe    /* read data */
#define ATAPI_POLL_DSC                  0xff    /* poll DSC status bit */


struct ata_ioc_devices {
    int                 channel;
    char                name[2][32];
    struct ata_params   params[2];
};

/* pr channel ATA ioctl calls */
#define IOCATAGMAXCHANNEL       _IOR('a',  1, int)
#define IOCATAREINIT            _IOW('a',  2, int)
#define IOCATAATTACH            _IOW('a',  3, int)
#define IOCATADETACH            _IOW('a',  4, int)
#define IOCATADEVICES           _IOWR('a',  5, struct ata_ioc_devices)

/* ATAPI request sense structure */
struct atapi_sense {
    u_int8_t	error;				/* current or deferred errors */
#define	ATA_SENSE_VALID			0x80

    u_int8_t	segment;			/* segment number */
    u_int8_t	key;				/* sense key */
#define ATA_SENSE_KEY_MASK		0x0f    /* sense key mask */
#define ATA_SENSE_NO_SENSE		0x00    /* no specific sense key info */
#define ATA_SENSE_RECOVERED_ERROR 	0x01    /* command OK, data recovered */
#define ATA_SENSE_NOT_READY		0x02    /* no access to drive */
#define ATA_SENSE_MEDIUM_ERROR		0x03    /* non-recovered data error */
#define ATA_SENSE_HARDWARE_ERROR	0x04    /* non-recoverable HW failure */
#define ATA_SENSE_ILLEGAL_REQUEST	0x05    /* invalid command param(s) */
#define ATA_SENSE_UNIT_ATTENTION	0x06    /* media changed */
#define ATA_SENSE_DATA_PROTECT		0x07    /* write protect */
#define ATA_SENSE_BLANK_CHECK		0x08    /* blank check */
#define ATA_SENSE_VENDOR_SPECIFIC	0x09    /* vendor specific skey */
#define ATA_SENSE_COPY_ABORTED		0x0a    /* copy aborted */
#define ATA_SENSE_ABORTED_COMMAND	0x0b    /* command aborted, try again */
#define ATA_SENSE_EQUAL			0x0c    /* equal */
#define ATA_SENSE_VOLUME_OVERFLOW	0x0d    /* volume overflow */
#define ATA_SENSE_MISCOMPARE		0x0e    /* data dont match the medium */
#define ATA_SENSE_RESERVED		0x0f
#define	ATA_SENSE_ILI			0x20;
#define	ATA_SENSE_EOM			0x40;
#define	ATA_SENSE_FILEMARK		0x80;

    u_int32_t   cmd_info;		/* cmd information */
    u_int8_t	sense_length;		/* additional sense len (n-7) */
    u_int32_t   cmd_specific_info;	/* additional cmd spec info */
    u_int8_t    asc;			/* additional sense code */
    u_int8_t    ascq;			/* additional sense code qual */
    u_int8_t    replaceable_unit_code;	/* replaceable unit code */
    u_int8_t	specific;		/* sense key specific */
#define	ATA_SENSE_SPEC_VALID	0x80
#define	ATA_SENSE_SPEC_MASK	0x7f
	
    u_int8_t	specific1;		/* sense key specific */
    u_int8_t	specific2;		/* sense key specific */
} __packed;

struct ata_ioc_request {
    union {
	struct {
	    u_int8_t            command;
	    u_int8_t            feature;
	    u_int64_t           lba;
	    u_int16_t           count;
	} ata;
	struct {
	    char                ccb[16];
	    struct atapi_sense	sense;
	} atapi;
    } u;
    caddr_t             data;
    int                 count;
    int                 flags;
#define ATA_CMD_CONTROL                 0x01
#define ATA_CMD_READ                    0x02
#define ATA_CMD_WRITE                   0x04
#define ATA_CMD_ATAPI                   0x08

    int                 timeout;
    int                 error;
};

struct ata_security_password {
	u_int16_t		ctrl;
#define ATA_SECURITY_PASSWORD_USER	0x0000
#define ATA_SECURITY_PASSWORD_MASTER	0x0001
#define ATA_SECURITY_ERASE_NORMAL	0x0000
#define ATA_SECURITY_ERASE_ENHANCED	0x0002
#define ATA_SECURITY_LEVEL_HIGH		0x0000
#define ATA_SECURITY_LEVEL_MAXIMUM	0x0100

	u_int8_t		password[32];
	u_int16_t		revision;
	u_int16_t		reserved[238];
};

/* pr device ATA ioctl calls */
#define IOCATAREQUEST           _IOWR('a', 100, struct ata_ioc_request)
#define IOCATAGPARM             _IOR('a', 101, struct ata_params)
#define IOCATAGMODE             _IOR('a', 102, int)
#define IOCATASMODE             _IOW('a', 103, int)

#define IOCATAGSPINDOWN		_IOR('a', 104, int)
#define IOCATASSPINDOWN		_IOW('a', 105, int)


struct ata_ioc_raid_config {
	    int                 lun;
	    int                 type;
#define AR_JBOD                         0x0001
#define AR_SPAN                         0x0002
#define AR_RAID0                        0x0004
#define AR_RAID1                        0x0008
#define AR_RAID01                       0x0010
#define AR_RAID3                        0x0020
#define AR_RAID4                        0x0040
#define AR_RAID5                        0x0080

	    int                 interleave;
	    int                 status;
#define AR_READY                        1
#define AR_DEGRADED                     2
#define AR_REBUILDING                   4

	    int                 progress;
	    int                 total_disks;
	    int                 disks[16];
};

struct ata_ioc_raid_status {
	    int                 lun;
	    int                 type;
	    int                 interleave;
	    int                 status;
	    int                 progress;
	    int                 total_disks;
	    struct {
		    int		state;
#define AR_DISK_ONLINE			0x01
#define AR_DISK_PRESENT			0x02
#define AR_DISK_SPARE			0x04
		    int		lun;
	    } disks[16];
};

/* ATA RAID ioctl calls */
#define IOCATARAIDCREATE        _IOWR('a', 200, struct ata_ioc_raid_config)
#define IOCATARAIDDELETE        _IOW('a', 201, int)
#define IOCATARAIDSTATUS        _IOWR('a', 202, struct ata_ioc_raid_status)
#define IOCATARAIDADDSPARE      _IOW('a', 203, struct ata_ioc_raid_config)
#define IOCATARAIDREBUILD       _IOW('a', 204, int)

#pragma clang diagnostic pop
