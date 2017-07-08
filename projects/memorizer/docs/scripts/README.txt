Files: memorizer.py, test_memorizer.py

Dependencies:
In order to run the test_memorizer w/ linux test suite, you must 
wget the latest version from the ltp github repo and set it up.
Ex:
wget https://github.com/linux-test-project/ltp/releases/download/20170116/ltp-full-20170116.tar.bz2
tar xvfj ltp-full-20170116.tar.bz2
# cd into the untarred dir
./configure
make
sudo make install

Good documentation / examples: http://ltp.sourceforge.net/documentation/how-to/ltp.php

memorizer.py: accepts processes to run in quotes. 
Ex: python memorizer.py "ls" "mkdir dir"
In order to run the script, you must have your user be in the 
memorizer group, which you should setup if not.
How-to: sudo groupadd memorizer; sudo usermod -aG memorizer <user>
You will be queried to enter your pw in order to set group 
permissions on the /sys/kernel/debug dirs which include ftrace
and memorizer.

test_memorizer.py: accepts either -e, -m, or -h flags.
Ex: python test_memorizer.py -e
*All modes will run the setup/cleanup checks to ensure all virtual nodes
are being set correctly.
-e: Runs ls, wget, and tar sequentially.
-m: Runs the linux test suite and saves a human-readable log to 
/opt/ltp/results/ltp.log
-h: runs both -e and -m
As with the memorizer.py, you will need your user to be in the
memorizer group.  Additionally, you will be queried to enter your
pw in order to set group permissions on the /opt/ltp dirs.

