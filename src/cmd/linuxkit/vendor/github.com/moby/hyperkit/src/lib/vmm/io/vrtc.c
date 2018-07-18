/*-
 * Copyright (c) 2014, Neel Natu (neel@freebsd.org)
 * Copyright (c) 2015 xhyve developers
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice unmodified, this list of conditions, and the following
 *    disclaimer.
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
 */

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>
#include <strings.h>
#include <pthread.h>
#include <errno.h>
#include <assert.h>
#include <mach/mach.h>
#include <mach/clock.h>
#include <xhyve/support/misc.h>
#include <xhyve/support/rtc.h>
#include <xhyve/vmm/vmm.h>
#include <xhyve/vmm/vmm_callout.h>
#include <xhyve/vmm/vmm_ktr.h>
#include <xhyve/vmm/io/vatpic.h>
#include <xhyve/vmm/io/vioapic.h>
#include <xhyve/vmm/io/vrtc.h>

static const u_char bin2bcd_data[] = {
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09,
	0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19,
	0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29,
	0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39,
	0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49,
	0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, 0x58, 0x59,
	0x60, 0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67, 0x68, 0x69,
	0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77, 0x78, 0x79,
	0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87, 0x88, 0x89,
	0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97, 0x98, 0x99
};

/* Register layout of the RTC */
struct rtcdev {
	uint8_t sec;
	uint8_t alarm_sec;
	uint8_t min;
	uint8_t alarm_min;
	uint8_t hour;
	uint8_t alarm_hour;
	uint8_t day_of_week;
	uint8_t day_of_month;
	uint8_t month;
	uint8_t year;
	uint8_t reg_a;
	uint8_t reg_b;
	uint8_t reg_c;
	uint8_t reg_d;
	uint8_t nvram[36];
	uint8_t century;
	uint8_t nvram2[128 - 51];
} __packed;
CTASSERT(sizeof(struct rtcdev) == 128);
CTASSERT(offsetof(struct rtcdev, century) == RTC_CENTURY);

struct vrtc {
	struct vm *vm;
	pthread_mutex_t mtx;
	struct callout callout;
	u_int addr; /* RTC register to read or write */
	sbintime_t base_uptime;
	time_t base_rtctime;
	struct rtcdev rtcdev;
};

struct clocktime {
	int	year; /* year (4 digit year) */
	int	mon; /* month (1 - 12) */
	int	day; /* day (1 - 31) */
	int	hour; /* hour (0 - 23) */
	int	min; /* minute (0 - 59) */
	int	sec; /* second (0 - 59) */
	int	dow; /* day of week (0 - 6; 0 = Sunday) */
	long nsec; /* nano seconds */
};

#define	VRTC_LOCK(vrtc) pthread_mutex_lock(&((vrtc)->mtx))
#define	VRTC_UNLOCK(vrtc) pthread_mutex_unlock(&((vrtc)->mtx))
/*
 * RTC time is considered "broken" if:
 * - RTC updates are halted by the guest
 * - RTC date/time fields have invalid values
 */
#define	VRTC_BROKEN_TIME	((time_t)-1)

#define	RTC_IRQ			8
#define	RTCSB_BIN		0x04
#define	RTCSB_ALL_INTRS		(RTCSB_UINTR | RTCSB_AINTR | RTCSB_PINTR)
#define	rtc_halted(vrtc)	((vrtc->rtcdev.reg_b & RTCSB_HALT) != 0)
#define	aintr_enabled(vrtc)	(((vrtc)->rtcdev.reg_b & RTCSB_AINTR) != 0)
#define	pintr_enabled(vrtc)	(((vrtc)->rtcdev.reg_b & RTCSB_PINTR) != 0)
#define	uintr_enabled(vrtc)	(((vrtc)->rtcdev.reg_b & RTCSB_UINTR) != 0)

static void vrtc_callout_handler(void *arg);
static void vrtc_set_reg_c(struct vrtc *vrtc, uint8_t newval);

static int rtc_flag_broken_time = 1;
static clock_serv_t mach_clock;

#define	POSIX_BASE_YEAR	1970
#define	FEBRUARY 2
#define SECDAY (24 * 60 * 60)
#define	days_in_year(y) (leapyear(y) ? 366 : 365)
#define	days_in_month(y, m) \
	(month_days[(m) - 1] + (m == FEBRUARY ? leapyear(y) : 0))
#define	day_of_week(days) (((days) + 4) % 7)

static const int month_days[12] = {
	31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31
};

static __inline int
leapyear(int year)
{
	int rv = 0;

	if ((year & 3) == 0) {
		rv = 1;
		if ((year % 100) == 0) {
			rv = 0;
			if ((year % 400) == 0)
				rv = 1;
		}
	}
	return (rv);
}

static int
clock_ct_to_ts(struct clocktime *ct, struct timespec *ts)
{
	int i, year, days;

	year = ct->year;

	/* Sanity checks. */
	if (ct->mon < 1 || ct->mon > 12 || ct->day < 1 ||
	    ct->day > days_in_month(year, ct->mon) ||
	    ct->hour > 23 ||  ct->min > 59 || ct->sec > 59 ||
	    (year > 2037 && sizeof(time_t) == 4)) {	/* time_t overflow */
		return (EINVAL);
	}

	/*
	 * Compute days since start of time
	 * First from years, then from months.
	 */
	days = 0;
	for (i = POSIX_BASE_YEAR; i < year; i++)
		days += days_in_year(i);

	/* Months */
	for (i = 1; i < ct->mon; i++)
		days += days_in_month(year, i);
	days += (ct->day - 1);

	ts->tv_sec = (((time_t)days * 24 + ct->hour) * 60 + ct->min) * 60 +
	    ct->sec;
	ts->tv_nsec = ct->nsec;

	return (0);
}

static void
clock_ts_to_ct(struct timespec *ts, struct clocktime *ct)
{
	int i, year, days;
	time_t rsec;	/* remainder seconds */
	time_t secs;

	secs = ts->tv_sec;
	days = (int) (secs / SECDAY);
	rsec = secs % SECDAY;

	ct->dow = day_of_week(days);

	/* Subtract out whole years, counting them in i. */
	for (year = POSIX_BASE_YEAR; days >= days_in_year(year); year++)
		days -= days_in_year(year);
	ct->year = year;

	/* Subtract out whole months, counting them in i. */
	for (i = 1; days >= days_in_month(year, i); i++)
		days -= days_in_month(year, i);
	ct->mon = i;

	/* Days are what is left over (+1) from all that. */
	ct->day = days + 1;

	/* Hours, minutes, seconds are easy */
	ct->hour = (int) (rsec / 3600);
	rsec = rsec % 3600;
	ct->min  = (int) (rsec / 60);
	rsec = rsec % 60;
	ct->sec  = (int) rsec;
	ct->nsec = ts->tv_nsec;
}

static __inline bool
divider_enabled(int reg_a)
{
	/*
	 * The RTC is counting only when dividers are not held in reset.
	 */
	return ((reg_a & 0x70) == 0x20);
}

static __inline bool
update_enabled(struct vrtc *vrtc)
{
	/*
	 * RTC date/time can be updated only if:
	 * - divider is not held in reset
	 * - guest has not disabled updates
	 * - the date/time fields have valid contents
	 */
	if (!divider_enabled(vrtc->rtcdev.reg_a))
		return (false);

	if (rtc_halted(vrtc))
		return (false);

	if (vrtc->base_rtctime == VRTC_BROKEN_TIME)
		return (false);

	return (true);
}

static time_t
vrtc_curtime(struct vrtc *vrtc, sbintime_t *basetime)
{
	sbintime_t now, delta;
	time_t t, secs;

	t = vrtc->base_rtctime;
	*basetime = vrtc->base_uptime;
	if (update_enabled(vrtc)) {
		now = sbinuptime();
		delta = now - vrtc->base_uptime;
		KASSERT(delta >= 0, ("vrtc_curtime: uptime went backwards: "
		    "%#llx to %#llx", vrtc->base_uptime, now));
		secs = delta / SBT_1S;
		t += secs;
		*basetime += secs * SBT_1S;
	}
	return (t);
}

static __inline uint8_t
rtcset(struct rtcdev *rtc, int val)
{

	KASSERT(val >= 0 && val < 100, ("%s: invalid bin2bcd index %d",
	    __func__, val));

	return ((uint8_t) ((rtc->reg_b & RTCSB_BIN) ? val : bin2bcd_data[val]));
}

static void
secs_to_rtc(time_t rtctime, struct vrtc *vrtc, int force_update)
{
	mach_timespec_t mts;
	struct clocktime ct;
	struct timespec ts;
	struct rtcdev *rtc;
	int hour;

	if (rtctime < 0) {
		KASSERT(rtctime == VRTC_BROKEN_TIME,
		    ("%s: invalid vrtc time %#lx", __func__, rtctime));
		return;
	}

	/*
	 * If the RTC is halted then the guest has "ownership" of the
	 * date/time fields. Don't update the RTC date/time fields in
	 * this case (unless forced).
	 */
	if (rtc_halted(vrtc) && !force_update)
		return;

	clock_get_time(mach_clock, &mts);
	ts.tv_sec = mts.tv_sec;
	ts.tv_nsec = mts.tv_nsec;

	clock_ts_to_ct(&ts, &ct);

	KASSERT(ct.sec >= 0 && ct.sec <= 59, ("invalid clocktime sec %d",
	    ct.sec));
	KASSERT(ct.min >= 0 && ct.min <= 59, ("invalid clocktime min %d",
	    ct.min));
	KASSERT(ct.hour >= 0 && ct.hour <= 23, ("invalid clocktime hour %d",
	    ct.hour));
	KASSERT(ct.dow >= 0 && ct.dow <= 6, ("invalid clocktime wday %d",
	    ct.dow));
	KASSERT(ct.day >= 1 && ct.day <= 31, ("invalid clocktime mday %d",
	    ct.day));
	KASSERT(ct.mon >= 1 && ct.mon <= 12, ("invalid clocktime month %d",
	    ct.mon));
	KASSERT(ct.year >= 1900, ("invalid clocktime year %d", ct.year));

	rtc = &vrtc->rtcdev;
	rtc->sec = rtcset(rtc, ct.sec);
	rtc->min = rtcset(rtc, ct.min);

	if (rtc->reg_b & RTCSB_24HR) {
		hour = ct.hour;
	} else {
		/*
		 * Convert to the 12-hour format.
		 */
		switch (ct.hour) {
		case 0:			/* 12 AM */
		case 12:		/* 12 PM */
			hour = 12;
			break;
		default:
			/*
			 * The remaining 'ct.hour' values are interpreted as:
			 * [1  - 11] ->  1 - 11 AM
			 * [13 - 23] ->  1 - 11 PM
			 */
			hour = ct.hour % 12;
			break;
		}
	}

	rtc->hour = rtcset(rtc, hour);

	if ((rtc->reg_b & RTCSB_24HR) == 0 && ct.hour >= 12)
		rtc->hour |= 0x80;	    /* set MSB to indicate PM */

	rtc->day_of_week = rtcset(rtc, ct.dow + 1);
	rtc->day_of_month = rtcset(rtc, ct.day);
	rtc->month = rtcset(rtc, ct.mon);
	rtc->year = rtcset(rtc, ct.year % 100);
	rtc->century = rtcset(rtc, ct.year / 100);
}

static int
rtcget(struct rtcdev *rtc, int val, int *retval)
{
	uint8_t upper, lower;

	if (rtc->reg_b & RTCSB_BIN) {
		*retval = val;
		return (0);
	}

	lower = val & 0xf;
	upper = (val >> 4) & 0xf;

	if (lower > 9 || upper > 9)
		return (-1);

	*retval = upper * 10 + lower;
	return (0);
}

static time_t
rtc_to_secs(struct vrtc *vrtc)
{
	struct clocktime ct;
	struct timespec ts;
	struct rtcdev *rtc;
	struct vm *vm;
	int century, error, hour, pm, year;

	vm = vrtc->vm;
	rtc = &vrtc->rtcdev;

	bzero(&ct, sizeof(struct clocktime));

	error = rtcget(rtc, rtc->sec, &ct.sec);
	if (error || ct.sec < 0 || ct.sec > 59) {
		VM_CTR2(vm, "Invalid RTC sec %#x/%d", rtc->sec, ct.sec);
		goto fail;
	}

	error = rtcget(rtc, rtc->min, &ct.min);
	if (error || ct.min < 0 || ct.min > 59) {
		VM_CTR2(vm, "Invalid RTC min %#x/%d", rtc->min, ct.min);
		goto fail;
	}

	pm = 0;
	hour = rtc->hour;
	if ((rtc->reg_b & RTCSB_24HR) == 0) {
		if (hour & 0x80) {
			hour &= ~0x80;
			pm = 1;
		}
	}
	error = rtcget(rtc, hour, &ct.hour);
	if ((rtc->reg_b & RTCSB_24HR) == 0) {
		if (ct.hour >= 1 && ct.hour <= 12) {
			/*
			 * Convert from 12-hour format to internal 24-hour
			 * representation as follows:
			 *
			 *    12-hour format		ct.hour
			 *	12	AM		0
			 *	1 - 11	AM		1 - 11
			 *	12	PM		12
			 *	1 - 11	PM		13 - 23
			 */
			if (ct.hour == 12)
				ct.hour = 0;
			if (pm)
				ct.hour += 12;
		} else {
			VM_CTR2(vm, "Invalid RTC 12-hour format %#x/%d",
			    rtc->hour, ct.hour);
			goto fail;
		}
	}

	if (error || ct.hour < 0 || ct.hour > 23) {
		VM_CTR2(vm, "Invalid RTC hour %#x/%d", rtc->hour, ct.hour);
		goto fail;
	}

	/*
	 * Ignore 'rtc->dow' because some guests like Linux don't bother
	 * setting it at all while others like OpenBSD/i386 set it incorrectly.
	 *
	 * clock_ct_to_ts() does not depend on 'ct.dow' anyways so ignore it.
	 */
	ct.dow = -1;

	error = rtcget(rtc, rtc->day_of_month, &ct.day);
	if (error || ct.day < 1 || ct.day > 31) {
		VM_CTR2(vm, "Invalid RTC mday %#x/%d", rtc->day_of_month,
		    ct.day);
		goto fail;
	}

	error = rtcget(rtc, rtc->month, &ct.mon);
	if (error || ct.mon < 1 || ct.mon > 12) {
		VM_CTR2(vm, "Invalid RTC month %#x/%d", rtc->month, ct.mon);
		goto fail;
	}

	error = rtcget(rtc, rtc->year, &year);
	if (error || year < 0 || year > 99) {
		VM_CTR2(vm, "Invalid RTC year %#x/%d", rtc->year, year);
		goto fail;
	}

	error = rtcget(rtc, rtc->century, &century);
	ct.year = century * 100 + year;
	if (error || ct.year < 1900) {
		VM_CTR2(vm, "Invalid RTC century %#x/%d", rtc->century,
		    ct.year);
		goto fail;
	}

	error = clock_ct_to_ts(&ct, &ts);
	if (error || ts.tv_sec < 0) {
		VM_CTR3(vm, "Invalid RTC clocktime.date %04d-%02d-%02d",
		    ct.year, ct.mon, ct.day);
		VM_CTR3(vm, "Invalid RTC clocktime.time %02d:%02d:%02d",
		    ct.hour, ct.min, ct.sec);
		goto fail;
	}
	return (ts.tv_sec);		/* success */
fail:
	/*
	 * Stop updating the RTC if the date/time fields programmed by
	 * the guest are invalid.
	 */
	VM_CTR0(vrtc->vm, "Invalid RTC date/time programming detected");
	return (VRTC_BROKEN_TIME);
}

static int
vrtc_time_update(struct vrtc *vrtc, time_t newtime, sbintime_t newbase)
{
	struct rtcdev *rtc;
	sbintime_t oldbase;
	time_t oldtime;
	uint8_t alarm_sec, alarm_min, alarm_hour;

	rtc = &vrtc->rtcdev;
	alarm_sec = rtc->alarm_sec;
	alarm_min = rtc->alarm_min;
	alarm_hour = rtc->alarm_hour;

	oldtime = vrtc->base_rtctime;
	VM_CTR2(vrtc->vm, "Updating RTC secs from %#lx to %#lx",
	    oldtime, newtime);

	oldbase = vrtc->base_uptime;
	VM_CTR2(vrtc->vm, "Updating RTC base uptime from %#llx to %#llx",
	    oldbase, newbase);
	vrtc->base_uptime = newbase;

	if (newtime == oldtime)
		return (0);

	/*
	 * If 'newtime' indicates that RTC updates are disabled then just
	 * record that and return. There is no need to do alarm interrupt
	 * processing in this case.
	 */
	if (newtime == VRTC_BROKEN_TIME) {
		vrtc->base_rtctime = VRTC_BROKEN_TIME;
		return (0);
	}

	/*
	 * Return an error if RTC updates are halted by the guest.
	 */
	if (rtc_halted(vrtc)) {
		VM_CTR0(vrtc->vm, "RTC update halted by guest");
		return (EBUSY);
	}

	do {
		/*
		 * If the alarm interrupt is enabled and 'oldtime' is valid
		 * then visit all the seconds between 'oldtime' and 'newtime'
		 * to check for the alarm condition.
		 *
		 * Otherwise move the RTC time forward directly to 'newtime'.
		 */
		if (aintr_enabled(vrtc) && oldtime != VRTC_BROKEN_TIME)
			vrtc->base_rtctime++;
		else
			vrtc->base_rtctime = newtime;

		if (aintr_enabled(vrtc)) {
			/*
			 * Update the RTC date/time fields before checking
			 * if the alarm conditions are satisfied.
			 */
			secs_to_rtc(vrtc->base_rtctime, vrtc, 0);

			if ((alarm_sec >= 0xC0 || alarm_sec == rtc->sec) &&
			    (alarm_min >= 0xC0 || alarm_min == rtc->min) &&
			    (alarm_hour >= 0xC0 || alarm_hour == rtc->hour)) {
				vrtc_set_reg_c(vrtc, rtc->reg_c | RTCIR_ALARM);
			}
		}
	} while (vrtc->base_rtctime != newtime);

	if (uintr_enabled(vrtc))
		vrtc_set_reg_c(vrtc, rtc->reg_c | RTCIR_UPDATE);

	return (0);
}

static sbintime_t
vrtc_freq(struct vrtc *vrtc)
{
	int ratesel;

	static sbintime_t pf[16] = {
		0,
		SBT_1S / 256,
		SBT_1S / 128,
		SBT_1S / 8192,
		SBT_1S / 4096,
		SBT_1S / 2048,
		SBT_1S / 1024,
		SBT_1S / 512,
		SBT_1S / 256,
		SBT_1S / 128,
		SBT_1S / 64,
		SBT_1S / 32,
		SBT_1S / 16,
		SBT_1S / 8,
		SBT_1S / 4,
		SBT_1S / 2,
	};

	/*
	 * If both periodic and alarm interrupts are enabled then use the
	 * periodic frequency to drive the callout. The minimum periodic
	 * frequency (2 Hz) is higher than the alarm frequency (1 Hz) so
	 * piggyback the alarm on top of it. The same argument applies to
	 * the update interrupt.
	 */
	if (pintr_enabled(vrtc) && divider_enabled(vrtc->rtcdev.reg_a)) {
		ratesel = vrtc->rtcdev.reg_a & 0xf;
		return (pf[ratesel]);
	} else if (aintr_enabled(vrtc) && update_enabled(vrtc)) {
		return (SBT_1S);
	} else if (uintr_enabled(vrtc) && update_enabled(vrtc)) {
		return (SBT_1S);
	} else {
		return (0);
	}
}

static void
vrtc_callout_reset(struct vrtc *vrtc, sbintime_t freqsbt)
{
	if (freqsbt == 0) {
		if (callout_active(&vrtc->callout)) {
			VM_CTR0(vrtc->vm, "RTC callout stopped");
			callout_stop(&vrtc->callout);
		}
		return;
	}
	VM_CTR1(vrtc->vm, "RTC callout frequency %lld hz", SBT_1S / freqsbt);
	callout_reset_sbt(&vrtc->callout, freqsbt, 0, vrtc_callout_handler,
	    vrtc, 0);
}

static void
vrtc_callout_handler(void *arg)
{
	struct vrtc *vrtc = arg;
	sbintime_t freqsbt, basetime;
	time_t rtctime;
	int error;

	VM_CTR0(vrtc->vm, "vrtc callout fired");

	VRTC_LOCK(vrtc);
	if (callout_pending(&vrtc->callout))	/* callout was reset */
		goto done;

	if (!callout_active(&vrtc->callout))	/* callout was stopped */
		goto done;

	callout_deactivate(&vrtc->callout);

	KASSERT((vrtc->rtcdev.reg_b & RTCSB_ALL_INTRS) != 0,
	    ("gratuitous vrtc callout"));

	if (pintr_enabled(vrtc))
		vrtc_set_reg_c(vrtc, vrtc->rtcdev.reg_c | RTCIR_PERIOD);

	if (aintr_enabled(vrtc) || uintr_enabled(vrtc)) {
		rtctime = vrtc_curtime(vrtc, &basetime);
		error = vrtc_time_update(vrtc, rtctime, basetime);
		KASSERT(error == 0, ("%s: vrtc_time_update error %d",
		    __func__, error));
	}

	freqsbt = vrtc_freq(vrtc);
	KASSERT(freqsbt != 0, ("%s: vrtc frequency cannot be zero", __func__));
	vrtc_callout_reset(vrtc, freqsbt);
done:
	VRTC_UNLOCK(vrtc);
}

static __inline void
vrtc_callout_check(struct vrtc *vrtc, sbintime_t freq)
{
	int active;

	active = callout_active(&vrtc->callout) ? 1 : 0;
	KASSERT((freq == 0 && !active) || (freq != 0 && active),
	    ("vrtc callout %s with frequency %#llx",
	    active ? "active" : "inactive", freq));
}

static void
vrtc_set_reg_c(struct vrtc *vrtc, uint8_t newval)
{
	struct rtcdev *rtc;
	int oldirqf, newirqf;
	uint8_t oldval, changed;

	rtc = &vrtc->rtcdev;
	newval &= RTCIR_ALARM | RTCIR_PERIOD | RTCIR_UPDATE;

	oldirqf = rtc->reg_c & RTCIR_INT;
	if ((aintr_enabled(vrtc) && (newval & RTCIR_ALARM) != 0) ||
	    (pintr_enabled(vrtc) && (newval & RTCIR_PERIOD) != 0) ||
	    (uintr_enabled(vrtc) && (newval & RTCIR_UPDATE) != 0)) {
		newirqf = RTCIR_INT;
	} else {
		newirqf = 0;
	}

	oldval = rtc->reg_c;
	rtc->reg_c = (uint8_t) (newirqf | newval);
	changed = oldval ^ rtc->reg_c;
	if (changed) {
		VM_CTR2(vrtc->vm, "RTC reg_c changed from %#x to %#x",
		    oldval, rtc->reg_c);
	}

	if (!oldirqf && newirqf) {
		VM_CTR1(vrtc->vm, "RTC irq %d asserted", RTC_IRQ);
		vatpic_pulse_irq(vrtc->vm, RTC_IRQ);
		vioapic_pulse_irq(vrtc->vm, RTC_IRQ);
	} else if (oldirqf && !newirqf) {
		VM_CTR1(vrtc->vm, "RTC irq %d deasserted", RTC_IRQ);
	}
}

static int
vrtc_set_reg_b(struct vrtc *vrtc, uint8_t newval)
{
	struct rtcdev *rtc;
	sbintime_t oldfreq, newfreq, basetime;
	time_t curtime, rtctime;
	int error;
	uint8_t oldval, changed;

	rtc = &vrtc->rtcdev;
	oldval = rtc->reg_b;
	oldfreq = vrtc_freq(vrtc);

	rtc->reg_b = newval;
	changed = oldval ^ newval;
	if (changed) {
		VM_CTR2(vrtc->vm, "RTC reg_b changed from %#x to %#x",
		    oldval, newval);
	}

	if (changed & RTCSB_HALT) {
		if ((newval & RTCSB_HALT) == 0) {
			rtctime = rtc_to_secs(vrtc);
			basetime = sbinuptime();
			if (rtctime == VRTC_BROKEN_TIME) {
				if (rtc_flag_broken_time)
					return (-1);
			}
		} else {
			curtime = vrtc_curtime(vrtc, &basetime);
			KASSERT(curtime == vrtc->base_rtctime, ("%s: mismatch "
			    "between vrtc basetime (%#lx) and curtime (%#lx)",
			    __func__, vrtc->base_rtctime, curtime));

			/*
			 * Force a refresh of the RTC date/time fields so
			 * they reflect the time right before the guest set
			 * the HALT bit.
			 */
			secs_to_rtc(curtime, vrtc, 1);

			/*
			 * Updates are halted so mark 'base_rtctime' to denote
			 * that the RTC date/time is in flux.
			 */
			rtctime = VRTC_BROKEN_TIME;
			rtc->reg_b &= ~RTCSB_UINTR;
		}
		error = vrtc_time_update(vrtc, rtctime, basetime);
		KASSERT(error == 0, ("vrtc_time_update error %d", error));
	}

	/*
	 * Side effect of changes to the interrupt enable bits.
	 */
	if (changed & RTCSB_ALL_INTRS)
		vrtc_set_reg_c(vrtc, vrtc->rtcdev.reg_c);

	/*
	 * Change the callout frequency if it has changed.
	 */
	newfreq = vrtc_freq(vrtc);
	if (newfreq != oldfreq)
		vrtc_callout_reset(vrtc, newfreq);
	else
		vrtc_callout_check(vrtc, newfreq);

	/*
	 * The side effect of bits that control the RTC date/time format
	 * is handled lazily when those fields are actually read.
	 */
	return (0);
}

static void
vrtc_set_reg_a(struct vrtc *vrtc, uint8_t newval)
{
	sbintime_t oldfreq, newfreq;
	uint8_t oldval, changed;

	newval &= ~RTCSA_TUP;
	oldval = vrtc->rtcdev.reg_a;
	oldfreq = vrtc_freq(vrtc);

	if (divider_enabled(oldval) && !divider_enabled(newval)) {
		VM_CTR2(vrtc->vm, "RTC divider held in reset at %#lx/%#llx",
		    vrtc->base_rtctime, vrtc->base_uptime);
	} else if (!divider_enabled(oldval) && divider_enabled(newval)) {
		/*
		 * If the dividers are coming out of reset then update
		 * 'base_uptime' before this happens. This is done to
		 * maintain the illusion that the RTC date/time was frozen
		 * while the dividers were disabled.
		 */
		vrtc->base_uptime = sbinuptime();
		VM_CTR2(vrtc->vm, "RTC divider out of reset at %#lx/%#llx",
		    vrtc->base_rtctime, vrtc->base_uptime);
	} else {
		/* NOTHING */
	}

	vrtc->rtcdev.reg_a = newval;
	changed = oldval ^ newval;
	if (changed) {
		VM_CTR2(vrtc->vm, "RTC reg_a changed from %#x to %#x",
		    oldval, newval);
	}

	/*
	 * Side effect of changes to rate select and divider enable bits.
	 */
	newfreq = vrtc_freq(vrtc);
	if (newfreq != oldfreq)
		vrtc_callout_reset(vrtc, newfreq);
	else
		vrtc_callout_check(vrtc, newfreq);
}

int
vrtc_set_time(struct vm *vm, time_t secs)
{
	struct vrtc *vrtc;
	int error;

	vrtc = vm_rtc(vm);
	VRTC_LOCK(vrtc);
	error = vrtc_time_update(vrtc, secs, sbinuptime());
	VRTC_UNLOCK(vrtc);

	if (error) {
		VM_CTR2(vrtc->vm, "Error %d setting RTC time to %#lx", error,
		    secs);
	} else {
		VM_CTR1(vrtc->vm, "RTC time set to %#lx", secs);
	}

	return (error);
}

time_t
vrtc_get_time(struct vm *vm)
{
	struct vrtc *vrtc;
	sbintime_t basetime;
	time_t t;

	vrtc = vm_rtc(vm);
	VRTC_LOCK(vrtc);
	t = vrtc_curtime(vrtc, &basetime);
	VRTC_UNLOCK(vrtc);

	return (t);
}

int
vrtc_nvram_write(struct vm *vm, int offset, uint8_t value)
{
	struct vrtc *vrtc;
	uint8_t *ptr;

	vrtc = vm_rtc(vm);

	/*
	 * Don't allow writes to RTC control registers or the date/time fields.
	 */
	if (((unsigned long) offset) < offsetof(struct rtcdev, nvram) ||
	    offset == RTC_CENTURY ||
	    ((unsigned long) offset) >= sizeof(struct rtcdev))
	{
		VM_CTR1(vrtc->vm, "RTC nvram write to invalid offset %d",
		    offset);
		return (EINVAL);
	}

	VRTC_LOCK(vrtc);
	ptr = (uint8_t *)(&vrtc->rtcdev);
	ptr[offset] = value;
	VM_CTR2(vrtc->vm, "RTC nvram write %#x to offset %#x", value, offset);
	VRTC_UNLOCK(vrtc);

	return (0);
}

int
vrtc_nvram_read(struct vm *vm, int offset, uint8_t *retval)
{
	struct vrtc *vrtc;
	sbintime_t basetime;
	time_t curtime;
	uint8_t *ptr;

	/*
	 * Allow all offsets in the RTC to be read.
	 */
	if (offset < 0 || ((unsigned long) offset) >= sizeof(struct rtcdev))
		return (EINVAL);

	vrtc = vm_rtc(vm);
	VRTC_LOCK(vrtc);

	/*
	 * Update RTC date/time fields if necessary.
	 */
	if (offset < 10 || offset == RTC_CENTURY) {
		curtime = vrtc_curtime(vrtc, &basetime);
		secs_to_rtc(curtime, vrtc, 0);
	}

	ptr = (uint8_t *)(&vrtc->rtcdev);
	*retval = ptr[offset];

	VRTC_UNLOCK(vrtc);
	return (0);
}

int
vrtc_addr_handler(struct vm *vm, UNUSED int vcpuid, bool in, UNUSED int port,
	int bytes, uint32_t *val)
{
	struct vrtc *vrtc;

	vrtc = vm_rtc(vm);

	if (bytes != 1)
		return (-1);

	if (in) {
		*val = 0xff;
		return (0);
	}

	VRTC_LOCK(vrtc);
	vrtc->addr = *val & 0x7f;
	VRTC_UNLOCK(vrtc);

	return (0);
}

int
vrtc_data_handler(struct vm *vm, int vcpuid, bool in, UNUSED int port,
	int bytes, uint32_t *val)
{
	struct vrtc *vrtc;
	struct rtcdev *rtc;
	sbintime_t basetime;
	time_t curtime;
	int error, offset;

	vrtc = vm_rtc(vm);
	rtc = &vrtc->rtcdev;

	if (bytes != 1)
		return (-1);

	VRTC_LOCK(vrtc);
	offset = (int) vrtc->addr;
	if (((unsigned long) offset) >= sizeof(struct rtcdev)) {
		VRTC_UNLOCK(vrtc);
		return (-1);
	}

	error = 0;
	curtime = vrtc_curtime(vrtc, &basetime);
	vrtc_time_update(vrtc, curtime, basetime);

	/*
	 * Update RTC date/time fields if necessary.
	 *
	 * This is not just for reads of the RTC. The side-effect of writing
	 * the century byte requires other RTC date/time fields (e.g. sec)
	 * to be updated here.
	 */
	if (offset < 10 || offset == RTC_CENTURY)
		secs_to_rtc(curtime, vrtc, 0);

	if (in) {
		if (offset == 12) {
			/*
			 * XXX
			 * reg_c interrupt flags are updated only if the
			 * corresponding interrupt enable bit in reg_b is set.
			 */
			*val = vrtc->rtcdev.reg_c;
			vrtc_set_reg_c(vrtc, 0);
		} else {
			*val = *((uint8_t *)rtc + offset);
		}
		VCPU_CTR2(vm, vcpuid, "Read value %#x from RTC offset %#x",
		    *val, offset);
	} else {
		switch (offset) {
		case 10:
			VCPU_CTR1(vm, vcpuid, "RTC reg_a set to %#x", *val);
			vrtc_set_reg_a(vrtc, ((uint8_t) *val));
			break;
		case 11:
			VCPU_CTR1(vm, vcpuid, "RTC reg_b set to %#x", *val);
			error = vrtc_set_reg_b(vrtc, ((uint8_t) *val));
			break;
		case 12:
			VCPU_CTR1(vm, vcpuid, "RTC reg_c set to %#x (ignored)",
			    *val);
			break;
		case 13:
			VCPU_CTR1(vm, vcpuid, "RTC reg_d set to %#x (ignored)",
			    *val);
			break;
		case 0:
			/*
			 * High order bit of 'seconds' is readonly.
			 */
			*val &= 0x7f;
			/* FALLTHRU */
		default:
			VCPU_CTR2(vm, vcpuid, "RTC offset %#x set to %#x",
			    offset, *val);
			*((uint8_t *)rtc + offset) = ((uint8_t) *val);
			break;
		}

		/*
		 * XXX some guests (e.g. OpenBSD) write the century byte
		 * outside of RTCSB_HALT so re-calculate the RTC date/time.
		 */
		if (offset == RTC_CENTURY && !rtc_halted(vrtc)) {
			curtime = rtc_to_secs(vrtc);
			error = vrtc_time_update(vrtc, curtime, sbinuptime());
			KASSERT(!error, ("vrtc_time_update error %d", error));
			if (curtime == VRTC_BROKEN_TIME && rtc_flag_broken_time)
				error = -1;
		}
	}
	VRTC_UNLOCK(vrtc);
	return (error);
}

void
vrtc_reset(struct vrtc *vrtc)
{
	struct rtcdev *rtc;

	VRTC_LOCK(vrtc);

	rtc = &vrtc->rtcdev;
	vrtc_set_reg_b(vrtc, rtc->reg_b & ~(RTCSB_ALL_INTRS | RTCSB_SQWE));
	vrtc_set_reg_c(vrtc, 0);
	KASSERT(!callout_active(&vrtc->callout), ("rtc callout still active"));

	VRTC_UNLOCK(vrtc);
}

struct vrtc *
vrtc_init(struct vm *vm)
{
	struct vrtc *vrtc;
	struct rtcdev *rtc;
	time_t curtime;

	vrtc = malloc(sizeof(struct vrtc));
	assert(vrtc);
	bzero(vrtc, sizeof(struct vrtc));
	vrtc->vm = vm;

	pthread_mutex_init(&vrtc->mtx, NULL);

	host_get_clock_service(mach_host_self(), CALENDAR_CLOCK, &mach_clock);

	callout_init(&vrtc->callout, 1);

	/* Allow dividers to keep time but disable everything else */
	rtc = &vrtc->rtcdev;
	rtc->reg_a = 0x20;
	rtc->reg_b = RTCSB_24HR;
	rtc->reg_c = 0;
	rtc->reg_d = RTCSD_PWR;

	/* Reset the index register to a safe value. */
	vrtc->addr = RTC_STATUSD;

	/*
	 * Initialize RTC time to 00:00:00 Jan 1, 1970.
	 */
	curtime = 0;

	VRTC_LOCK(vrtc);
	vrtc->base_rtctime = VRTC_BROKEN_TIME;
	vrtc_time_update(vrtc, curtime, sbinuptime());
	secs_to_rtc(curtime, vrtc, 0);
	VRTC_UNLOCK(vrtc);

	return (vrtc);
}

void
vrtc_cleanup(struct vrtc *vrtc)
{
	callout_drain(&vrtc->callout);
	mach_port_deallocate(mach_task_self(), mach_clock);
	free(vrtc);
}
