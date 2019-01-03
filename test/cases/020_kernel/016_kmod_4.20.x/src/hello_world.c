/*  
 * A simple Hello World kernel module 
 */
#include <linux/module.h>
#include <linux/kernel.h>

int init_hello(void)
{
	printk(KERN_INFO "Hello LinuxKit\n");
	return 0;
}

void exit_hello(void)
{
	printk(KERN_INFO "Goodbye LinuxKit.\n");
}

module_init(init_hello);
module_exit(exit_hello);
MODULE_AUTHOR("Rolf Neugebauer <rolf.neugebauer@docker.com>");
MODULE_LICENSE("GPL");
MODULE_DESCRIPTION("A simple Hello World kernel module for testing");
