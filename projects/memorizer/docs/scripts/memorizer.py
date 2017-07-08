import sys,threading,os,subprocess,operator,time

mem_path = "/sys/kernel/debug/memorizer/"
directory = ""
completed = False

def worker(cmd):
  ret = os.system(cmd)    
  if(ret != 0):
    print "Failed attempt on: " + cmd
    exit(1)

def basic_cleanup():
  print "Basic tests completed. Now cleaning up."
  ret = os.system("rm UPennlogo2.jpg")
        
def memManager():
  while(not completed):
    stats = subprocess.check_output(["free"])
    stats_list = stats.split()
    total_mem = float(stats_list[7])
    used_mem = float(stats_list[8])
    memory_usage = used_mem / total_mem
    if(memory_usage > 0.8):
      ret = os.system("cat " + mem_path + "kmap >> " + directory + "test.kmap")
      if ret != 0:
        print "Failed to append kmap to temp file"
        exit(1)
      ret = os.system("echo 1 > " + mem_path + "clear_printed_list")
      if ret != 0:
        print "Failed to clear printed list"
        exit(1)
    time.sleep(2)
            
def startup():
  ret = os.system("sudo chgrp -R memorizer /opt/")
  if ret != 0:
    print "Failed to change group permissions of /opt/"
    exit(1)
  os.system("sudo chmod -R g+wrx /opt/")
  if ret != 0:
    print "Failed to grant wrx permissions to /opt/"
    exit(1)
  # Setup group permissions to ftrace & memorizer directories
  ret = os.system("sudo chgrp -R memorizer /sys/kernel/debug/")
  if ret != 0:
    print "Failed to change memorizer group permissions to /sys/kernel/debug/"
    exit(1)
  ret = os.system("sudo chmod -R g+wrx /sys/kernel/debug/")
  if ret != 0:
    print "Failed to grant wrx persmissions to /sys/kernel/debug/"
    exit(1)
  # Memorizer Startup
  ret = os.system("echo 1 > " + mem_path + "clear_object_list")
  if ret != 0:
    print "Failed to clear object list"
    exit(1)
  ret = os.system("echo 0 > " + mem_path + "print_live_obj")
  if ret != 0:
    print "Failed to disable live object dumping"
    exit(1)
  ret = os.system("echo 1 > " + mem_path + "memorizer_enabled")
  if ret != 0:
    print "Failed to enable memorizer object allocation tracking"
    exit(1)
  ret = os.system("echo 1 > " + mem_path + "memorizer_log_access")
  if ret != 0:
    print "Failed to enable memorizer object access tracking"
    exit(1)

def cleanup():
  # Memorizer cleanup
  ret = os.system("echo 0 > " + mem_path + "memorizer_log_access")
  if ret != 0:
    print "Failed to disable memorizer object access tracking"
    exit(1)
  ret = os.system("echo 0 > " + mem_path + "memorizer_enabled")
  if ret != 0:
    print "Failed to disable memorizer object allocation tracking"
    exit(1)
  # Print stats
  ret = os.system("cat " + mem_path + "show_stats")
  if ret != 0:
    print "Failed to display memorizer stats"
    exit(1)
  ret = os.system("echo 1 > " + mem_path + "print_live_obj")
  if ret != 0:
    print "Failed to enable live object dumping"
    exit(1)
  # Make local copies of outputs
  ret = os.system("cat " + mem_path + "kmap >> " +directory+ "test.kmap")
  if ret != 0:
    print "Failed to copy live and freed objs to kmap"
    exit(1)
  ret = os.system("echo 1 > " + mem_path + "clear_object_list")
  if ret != 0:
    print "Failed to clear all freed objects in obj list"
    exit(1)

def main(argv):
  global completed
  global directory
  if len(sys.argv) == 1:
    print "Invalid/missing arg. Please enter -e for basic tests, -m for ltp tests, and/or specify a full process to run in quotes. Specify path using the -p <path> otherwise default to ."
    return
  startup()
  processes = []
  easy_processes = False
  next_arg = False
  for arg in argv:
    if next_arg: 
      next_arg = False
      directory = str(arg) + "/"
    elif arg == '-p':
      next_arg = True
    #User wants to run ltp
    elif arg == '-m':
      print "Performing ltp tests" 
      processes.append("/opt/ltp/runltp -p -l ltp.log")
      print "See /opt/ltp/results/ltp.log for ltp results"
    #User wants to run wget,ls,etc.
    elif arg == '-e':
      easy_processes = True
      print "Performing basic ls test"
      processes.append("ls")
      print "Performing wget test"
      processes.append("wget http://www.sas.upenn.edu/~egme/UPennlogo2.jpg")
  print "Attempting to remove any existing kmaps in the specified path"
  os.system("rm " + directory + "test.kmap")
  print "Startup completed. Generating threads."
  manager = threading.Thread(target=memManager, args=())
  manager.start()
  threads = []
  for process in processes:
    try:
      t = threading.Thread(target=worker, args=(process,))
      threads.append(t)
      t.start()
    except:
      print "Error: unable to start thread"
  for thr in threads:
    thr.join()
  completed = True
  manager.join()
  print "Threads ran to completion. Cleaning up."
  basic_cleanup()
  cleanup()
  print "Cleanup successful."
  return 0

if __name__ == "__main__":
  main(sys.argv)
