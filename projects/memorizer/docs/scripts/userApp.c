#include <stdio.h>
#include <sys/syscall.h>
#include <unistd.h>
#include <stdio.h>
#include <unistd.h>
#include <sys/mman.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <stdlib.h>
#include <stdint.h>
#include <inttypes.h>
#include "event_structs.h"


#define ML 400000  // The size of profiler buffer (Unit: memory page)

#define BUFF_MUTEX_LOCK { \
		while(*buff_mutex); \
		*buff_mutex = *buff_mutex + 1;\
	}

#define BUFF_MUTEX_UNLOCK {*buff_mutex = *buff_mutex - 1;}

#define BUFF_FILL_RESET {*buff_fill = 0;}



static int buf_fd = -1;
static int buf_len;
struct stat s ;
char *buf;
char *buff_end;
char *buff_fill;
struct memorizer_kernel_event *mke_ptr;
unsigned int *buff_free_size; 



// This function opens a character device (which is pointed by a file named as fname) and performs the mmap() operation. If the operations are successful, the base address of memory mapped buffer is returned. Otherwise, a NULL pointer is returned.
void *buf_init(char *fname)
{
	unsigned int *kadr;

	if(buf_fd == -1){
	buf_len = ML * getpagesize();
	if ((buf_fd=open(fname, O_RDWR|O_SYNC))<0){
	          printf("File open error. %s\n", fname);
	          return NULL;
		}
	}
	kadr = mmap(0, buf_len, PROT_READ|PROT_WRITE, MAP_SHARED, buf_fd, 0);
	if (kadr == MAP_FAILED){
		printf("Buf file open error.\n");
		return NULL;
		}
	return kadr;
}

// This function closes the opened character device file
void buf_exit()
{
	if(buf_fd!=-1){
		close(buf_fd);
		buf_fd = -1;
	}
}

void printAllocHex()
{
	struct memorizer_kernel_alloc *mke_ptr;
	mke_ptr = (struct memorizer_kernel_alloc *)buf;
	printf("aa, ");
	printf("%llx, ",(unsigned long long)mke_ptr->event_ip);
	printf("%llx, ",(unsigned long long)mke_ptr->src_va_ptr);
	printf("%llx, ",(unsigned long long)mke_ptr->src_pa_ptr);
	printf("%x, ",mke_ptr->event_size);
	printf("%lx, ",mke_ptr->event_jiffies);	
	printf("%x, ",mke_ptr->pid);
	printf("%s, ",mke_ptr->comm);
	printf("%s\n",mke_ptr->funcstr);
	buf = buf + sizeof(struct memorizer_kernel_alloc);
}

void printAlloc()
{
	struct memorizer_kernel_alloc *mke_ptr;
	mke_ptr = (struct memorizer_kernel_alloc *)buf;
	printf("Alloc: ");
	printf("%llx, ",(unsigned long long)mke_ptr->event_ip);
	printf("%llx, ",(unsigned long long)mke_ptr->src_va_ptr);
	printf("%llx, ",(unsigned long long)mke_ptr->src_pa_ptr);
	printf("%u, ",mke_ptr->event_size);
	printf("%lu, ",mke_ptr->event_jiffies);	
	printf("%u, ",mke_ptr->pid);
	printf("%s, ",mke_ptr->comm);
	printf("%s\n",mke_ptr->funcstr);
	buf = buf + sizeof(struct memorizer_kernel_alloc);
}


void printFreeHex()
{
	struct memorizer_kernel_free *mke_ptr;
	mke_ptr = (struct memorizer_kernel_free *)buf;
	printf("0xbb, ");
	printf("%llx, ",(unsigned long long)mke_ptr->event_ip);
	printf("%llx, ",(unsigned long long)mke_ptr->src_va_ptr);
	printf("%lx, ",mke_ptr->event_jiffies);	
	printf("%x\n",mke_ptr->pid);
	buf = buf + sizeof(struct memorizer_kernel_free);
}

void printFree()
{
	struct memorizer_kernel_free *mke_ptr;
	mke_ptr = (struct memorizer_kernel_free *)buf;
	printf("Free: ");
	printf("%llx, ",(unsigned long long)mke_ptr->event_ip);
	printf("%llx, ",(unsigned long long)mke_ptr->src_va_ptr);
	printf("%lu, ",mke_ptr->event_jiffies);	
	printf("%u\n",mke_ptr->pid);
	buf = buf + sizeof(struct memorizer_kernel_free);
}

void printAccessHex(char type)
{
	struct memorizer_kernel_access *mke_ptr;
	mke_ptr = (struct memorizer_kernel_access *)buf;
	if(type=='r')
		printf("0xcc, ");
	else
		printf("0xdd, ");
	printf("%llx, ",(unsigned long long)mke_ptr->event_ip);
	printf("%llx, ",(unsigned long long)mke_ptr->src_va_ptr);
	printf("%x, ",mke_ptr->event_size);
	printf("%lx, ",mke_ptr->event_jiffies);	
	printf("%x\n",mke_ptr->pid);
	buf = buf + sizeof(struct memorizer_kernel_access);
}


void printAccess(char type)
{
	struct memorizer_kernel_access *mke_ptr;
	mke_ptr = (struct memorizer_kernel_access *)buf;
	if(type=='r')
		printf("Read: ");
	else
		printf("Write: ");
	printf("%llx, ",(unsigned long long)mke_ptr->event_ip);
	printf("%llx, ",(unsigned long long)mke_ptr->src_va_ptr);
	printf("%u, ",mke_ptr->event_size);
	printf("%lu, ",mke_ptr->event_jiffies);	
	printf("%u\n",mke_ptr->pid);
	buf = buf + sizeof(struct memorizer_kernel_access);
}

void printFork()
{
	struct memorizer_kernel_fork *mke_ptr;
	mke_ptr = (struct memorizer_kernel_fork *)buf;
	printf("Fork: ");
	printf("%ld, ",mke_ptr->pid);
	printf("%s\n",mke_ptr->comm);
	buf = buf + sizeof(struct memorizer_kernel_fork);

}

int main (int argc, char *argv[])
{
	if(argc != 2)
	{
		printf("Incorrect number of Command Line Arguments!\n");
		return 0;
	}

	// Open the Character Device and MMap 
	buf = buf_init("node");
	if(!buf)
		return -1;

	//Read and count the MMaped data entries
	buff_end = (buf + ML*getpagesize()) - 1;
	buff_fill = buf;
	buf++;
	buff_free_size = (unsigned int *)buf;
	buf = buf + sizeof(unsigned int);

	mke_ptr = (struct memorizer_kernel_event *)buf;
	if(*argv[1]=='c')
	{
		printf("Remaining Bytes: ");
		printf("%u",*buff_free_size);
	}
	else if(*argv[1]=='p')
	{
	
		//TODO: Call different functions for different events
		while(*buf!=0)
		{
			if(*buf == 0xffffffaa)
				printAlloc();
			else if (*buf == 0xffffffbb)
				printFree();
			else if(*buf == 0xffffffcc)
				printAccess('r');
			else if(*buf == 0xffffffdd)
				printAccess('w');
			else if(*buf == 0xffffffee)
				printFork();

		}	

	}
	else if(*argv[1]=='h')
	{
	
		//TODO: Call different functions for different events
		while(*buf!=0)
		{
			if(*buf == 0xffffffaa)
				printAllocHex();
			else if (*buf == 0xffffffbb)
				printFreeHex();
			else if(*buf == 0xffffffcc)
				printAccessHex('r');
			else if(*buf == 0xffffffdd)
				printAccessHex('w');
		}	

	}	
	buf_exit();
	
	return 0;
}

