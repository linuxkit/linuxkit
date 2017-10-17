/* This file describes the structs to be used to describe the events happening inside the kernel:
 * 1. ALLOCATIONS
 * 2. DEALLOCATIONS
 * 3. ACCESSES
 * These will be used to create stateless logs for Memorizer 2.0
 * */

#include <linux/sched.h>

/* Event and Access type  enumerations */
enum EventType {Memorizer_Mem_Alloc = 0xaa, Memorizer_Mem_Free = 0xbb, Memorizer_Mem_Access = 0xcc};
enum AccessType {Memorizer_READ=0,Memorizer_WRITE};

struct memorizer_kernel_event {
	enum EventType event_type;
	uintptr_t	event_ip;
	uintptr_t	src_va_ptr;
	uintptr_t	src_pa_ptr;
	size_t		event_size;
	unsigned long	event_jiffies;
	pid_t		pid;
	enum		AccessType access_type;
	char		comm[16];
	char		funcstr[128];

			
};

struct memorizer_kernel_alloc {
	char		event_type;
	uintptr_t	event_ip;
	uintptr_t	src_va_ptr;
	uintptr_t	src_pa_ptr;
	size_t		event_size;
	unsigned long	event_jiffies;
	pid_t		pid;
	char		comm[16];
	char		funcstr[128];
};

struct memorizer_kernel_free {
	char		event_type;
	uintptr_t	event_ip;
	uintptr_t	src_va_ptr;
	unsigned long	event_jiffies;
	pid_t		pid;
};

struct memorizer_kernel_access {
	char		event_type;
	uintptr_t	event_ip;
	uintptr_t	src_va_ptr;
	size_t		event_size;
	unsigned long	event_jiffies;
	pid_t		pid;
};

struct memorizer_kernel_fork {
	char		event_type;
	long		pid;
	char		comm[16];
};
