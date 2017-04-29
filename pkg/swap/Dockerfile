FROM alpine:3.5

# swap command - only minimal Alpine install

# use util-linux to get swapfile utils, cryptsetup
#   at some point, hopefully use our own mkswap and swapon, as well as cryptsetup
RUN apk add --update util-linux cryptsetup

# add the entrypoint and make it executable
COPY . ./

# ENTRYPOINT
ENTRYPOINT swap.sh
