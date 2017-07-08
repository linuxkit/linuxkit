# Memorizer

Memorizer is a tool to trace fine-grained intra-kernel
operations. The goal is to track interactions with memory
objects for the purpose of analyzing fine-grained
interactions amongst components and execution contexts.
Memorizer tracks the following object operations: creation
(alloc), destruction (free), modify (store), access (load),
call, and return. 

Nathan D. ([@ndauten]) presented the umbrella project,
Opportunistic Privilege Separation (OPS), and Memorizer at
the [7/9/17 LinuxKit SIG](../../reports/2017-07-09.md) and
[slides](http://nathandautenhahn.com/talks/2017-06-21_ops+memorizer-linuxkit-sig/linuxkit-sig-remark.html#1)

## Usage

See [manual usage docs](./docs/memorizer.txt). Be careful
though because if the event queues are not drained then the
system will run out of memory. 

For controlled use see [script + readme](./docs/memorizer/).
This script is not automatically inserted into the runtime
yet.

## Issues

- KASAN is reporting some errors within itself. This is
  noisy. Can reduce the console log output level to < 3,
  e.g., `echo 3 > /proc/sys/kernel/printk`

- Source should be included soon, but for now there is an
  image on Docker Hub. 
