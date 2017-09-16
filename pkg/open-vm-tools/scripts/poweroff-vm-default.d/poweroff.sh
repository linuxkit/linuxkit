#!/bin/sh
# send SIGTERM to the init system (PID 1), which causes a clean VM host-initiated shutdown
kill -s SIGTERM 1
exit 0
