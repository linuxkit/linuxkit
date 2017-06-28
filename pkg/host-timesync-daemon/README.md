### host-timesync-daemon

Some hypervisors (e.g. hyperkit / xhyve) don't provide a good way to keep
the VM's clock in sync with the Host's clock. NTP will usually keep the
clocks together, but after the host or VM is suspended and resumed the
clocks can be suddenly too far apart for NTP to work properly.

This simple daemon listens on an AF_VSOCK port. When a connection is
received, the daemon

- reads the (hypervisor's virtual) hardware clock via `RTC_RD_TIME` on
  `/dev/rtc0`
- calls `settimeofday` to set the time in the VM
- closes the connection

Note the hardware clock has only second granularity so you will still need
NTP (or some other process) to keep the clocks closely synchronised.

To use this, simply connect to the AF_VSOCK port when you want the clock
to be resynchronised. If you want to wait for completion then `read` from
the socket -- `EOF` means the resychronisation is complete.
