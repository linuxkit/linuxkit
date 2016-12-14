#include <sys/types.h>
#include <errno.h>

#include <stdlib.h>
#include <string.h>
#include <syslog.h>
#include <libgen.h>

#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>
#include <dirent.h>
#include <sys/socket.h>
#include <sys/wait.h>
#include <pthread.h>
#include <signal.h>

#include <netinet/in.h>
#include <inttypes.h>
#include <sys/stat.h>

#include <sys/time.h>
#include <sys/resource.h>

#include "transfused_log.h"
#include "transfused_vsock.h"

char *default_fusermount = DEFAULT_FUSERMOUNT;
char *default_socket = DEFAULT_SOCKET;
char *default_server = DEFAULT_SERVER;
char *usage =
"usage: transfused [-p pidfile] [-d server] [-s socket] [-f fusermount]\n"
"                  [-l logfile]\n"
" -p pidfile\tthe path at which to write the pid of the process\n"
" -d " DEFAULT_SERVER "\tthe server address to use ('v:addr:port')\n"
" -s " DEFAULT_SOCKET "\tthe socket address to use ('v:addr:port')\n"
" -f " DEFAULT_FUSERMOUNT "\tthe fusermount executable to use\n"
" -l logfile\tthe log file to use before uplink\n";

int debug;

pthread_attr_t detached;

typedef struct {
	connection_t *connection;
	int from;
	int to;
} copy_thread_state;

#include <sys/syscall.h>

pid_t gettid(void)
{
	return syscall(SYS_gettid);
}

void *must_malloc(char *const descr, size_t size)
{
	void *ptr;

	ptr = malloc(size);
	if (size != 0 && ptr == NULL)
		die(1, NULL, descr, "");

	return ptr;
}

void cond_init(char *const descr, pthread_cond_t *cond,
	       const pthread_condattr_t *restrict attr)
{
	errno = pthread_cond_init(cond, attr);
	if (errno)
		die(1, NULL, "", "cond init %s: ", descr);
}

void lock_init(char *const descr, pthread_mutex_t *mutex,
	       const pthread_mutexattr_t *restrict attr)
{
	errno = pthread_mutex_init(mutex, attr);
	if (errno)
		die(1, NULL, "", "lock init %s: ", descr);
}

void lock(char *const descr, pthread_mutex_t *mutex)
{
	errno = pthread_mutex_lock(mutex);
	if (errno)
		die(1, NULL, "", "lock %s: ", descr);
}

void unlock(char *const descr, pthread_mutex_t *mutex)
{
	errno = pthread_mutex_unlock(mutex);
	if (errno)
		die(1, NULL, "", "unlock %s: ", descr);
}

int bind_socket(const char *socket)
{
	int sock;

	if (socket[0] == 0)
		die(2, NULL, NULL, "Socket family required");

	if (socket[1] != ':')
		die(2, NULL, NULL, "Socket address required");

	switch (socket[0]) {
	case 'v':
		sock = bind_vsock(socket + 2);
		break;
	default:
		die(2, NULL, NULL, "Unknown socket family '%c'", socket[0]);
	}

	return sock;
}

int connect_socket(const char *socket)
{
	int sock;

	if (socket[0] == 0)
		die(2, NULL, NULL, "Socket family required");
	if (socket[1] != ':')
		die(2, NULL, NULL, "Scoket address required");

	switch (socket[0]) {
	case 'v':
		sock = connect_vsock(socket + 2);
		break;
	default:
		die(2, NULL, NULL, "Unknown socket family '%c'", socket[0]);
	}

	return sock;
}

char **read_opts(connection_t *conn, char *buf)
{
	int read_count;
	int optc = 1;
	char **optv;
	size_t mount_len;
	int j;

	/* TODO: deal with socket read conditions e.g.EAGAIN */
	read_count = read(conn->sock, buf, EVENT_BUFSZ - 1);
	if (read_count < 0)
		die(1, conn->params, "read_opts error reading", "");

	/* TODO: protocol should deal with short read */
	buf[read_count] = 0x0;

	for (int i = 0; i < read_count; i++) {
		if (buf[i] == 0x0)
			optc++;
	}

	optv = (char **)must_malloc("read_opts optv", (optc + 1) * sizeof(void *));
	optv[0] = buf;
	optv[optc] = 0x0;

	j = 1;
	for (int i = 0; i < read_count && j < optc; i++) {
		if (buf[i] == 0x0) {
			optv[j] = buf + i + 1;
			j++;
		}
	}

	mount_len = strnlen(optv[optc - 1], 4096) + 1;
	conn->mount_point = must_malloc("mount point string", mount_len);
	strncpy(conn->mount_point, optv[optc - 1], mount_len - 1);
	conn->mount_point[mount_len - 1] = '\0';

	return optv;
}

uint64_t message_id(uint64_t *message)
{
	return message[1];
}

void read_exactly(char *descr, int fd, void *p, size_t nbyte)
{
	ssize_t read_count;
	char *buf = p;

	while (nbyte > 0) {
		read_count = read(fd, buf, nbyte);
		if (read_count < 0) {
			if (errno == EAGAIN || errno == EINTR)
				continue;
			die(1, NULL, "", "read %s: error reading: ", descr);
		}
		if (read_count == 0)
			die(1, NULL, NULL, "read %s: EOF reading", descr);
		nbyte -= read_count;
		buf += read_count;
	}
}

int read_message(char *descr, parameters *params, int fd,
		 char *buf, size_t max_read)
{
	size_t nbyte = sizeof(uint32_t);
	uint32_t len;

	read_exactly(descr, fd, buf, nbyte);
	len = *((uint32_t *) buf);
	if (len > max_read)
		die(1, params, NULL,
		    "read %s: message size %d exceeds buffer capacity %d",
		    len, max_read);
	if (len < nbyte)
		die(1, params, NULL,
		    "read %s: message size is %d but must be at least %d",
		    len, nbyte);

	buf += nbyte;
	nbyte = (size_t)(len - nbyte);

	read_exactly(descr, fd, buf, nbyte);

	return (int)len;
}

void copy_into_fuse(copy_thread_state *copy_state)
{
	int from = copy_state->from;
	int to = copy_state->to;
	char *descr = copy_state->connection->mount_point;
	int read_count, write_count;
	void *buf;
	parameters *params = copy_state->connection->params;

	buf = must_malloc(descr, IN_BUFSZ);

	while (1) {
		read_count = read_message(descr, params,
					  from, (char *)buf, IN_BUFSZ);

		write_count = write(to, buf, read_count);
		if (write_count < 0)
			die(1, params, "", "copy %s: error writing: ", descr);

		/* /dev/fuse accepts only complete writes */
		if (write_count != read_count)
			die(1, params, NULL,
			    "copy %s: read %d but only wrote %d",
			    descr, read_count, write_count);
	}

	free(buf);
}

void copy_notify_fuse(copy_thread_state *copy_state)
{
	int from = copy_state->from;
	int to = copy_state->to;
	char *descr = copy_state->connection->mount_point;
	int read_count, write_count;
	uint32_t zero = 0, err;
	void *buf;
	parameters *params = copy_state->connection->params;

	buf = must_malloc(descr, IN_BUFSZ);

	while (1) {
		read_count = read_message(descr, params,
					  from, (char *)buf, IN_BUFSZ);
		write_count = write(to, buf, read_count);
		if (write_count < 0) {
			err = errno;
			write_count = write(from, &err, 4);
			if (write_count < 0) {
				log_time(params,
					 "copy notify %s write error: %s", strerror(err));
				die(1, params, "",
				    "copy notify %s reply write error: ", descr);
			}
			continue;
		} else {
			if (write(from, &zero, 4) < 0)
				die(1, params, "",
				    "copy notify %s reply write error: ", descr);
		}

		if (write_count != read_count)
			die(1, params, NULL,
			    "copy notify %s: read %d but only wrote %d",
			    descr, read_count, write_count);
	}

	free(buf);
}

void write_exactly(char *descr, int fd, void *p, size_t nbyte)
{
	int write_count;
	char *buf = p;

	while (nbyte > 0) {
		write_count = write(fd, buf, nbyte);
		if (write_count < 0) {
			if (errno == EINTR || errno == EAGAIN)
				continue;
			die(1, NULL, "", "%s: error writing: ", descr);
		}
		if (write_count == 0)
			die(1, NULL, "", "%s: 0 write: ", descr);

		nbyte -= write_count;
		buf += write_count;
	}
}

void copy_outof_fuse(copy_thread_state *copy_state)
{
	int from = copy_state->from;
	int to = copy_state->to;
	char *descr = copy_state->connection->mount_point;
	int read_count;
	void *buf;
	parameters *params = copy_state->connection->params;

	buf = must_malloc(descr, OUT_BUFSZ);

	while (1) {
		/* /dev/fuse only returns complete reads */
		read_count = read(from, buf, OUT_BUFSZ);
		if (read_count < 0)
			die(1, params, "", "copy %s: error reading: ", descr);

		write_exactly(descr, to, (char *)buf, read_count);
	}

	free(buf);
}

void *copy_clean_into_fuse(copy_thread_state *copy_state)
{
	copy_into_fuse(copy_state);

	close(copy_state->from);

	free(copy_state);

	return NULL;
}

void *copy_clean_into_fuse_thread(void *copy_state)
{
	return copy_clean_into_fuse((copy_thread_state *)copy_state);
}

void *copy_clean_notify_fuse(copy_thread_state *copy_state)
{
	copy_notify_fuse(copy_state);

	close(copy_state->from);

	free(copy_state);

	return NULL;
}

void *copy_clean_notify_fuse_thread(void *copy_state)
{
	return copy_clean_notify_fuse((copy_thread_state *) copy_state);
}

void *copy_clean_outof_fuse(copy_thread_state *copy_state)
{
	copy_outof_fuse(copy_state);

	close(copy_state->to);

	free(copy_state);

	return NULL;
}

void *copy_clean_outof_fuse_thread(void *copy_state)
{
	return copy_clean_outof_fuse((copy_thread_state *) copy_state);
}

int recv_fd(parameters *params, int sock)
{
	int ret;
	int fd = -1;
	char iochar;
	char buf[CMSG_SPACE(sizeof(fd))];

	struct msghdr msg;
	struct iovec vec;
	struct cmsghdr *cmsg;

	msg.msg_name = NULL;
	msg.msg_namelen = 0;
	vec.iov_base = &iochar;
	vec.iov_len = 1;
	msg.msg_iov = &vec;

	msg.msg_iovlen = 1;

	msg.msg_control = buf;
	msg.msg_controllen = sizeof(buf);

	ret = recvmsg(sock, &msg, 0);

	if (ret == -1)
		die(1, params, "recvmsg", "");

	if (ret > 0 && msg.msg_controllen > 0) {
		cmsg = CMSG_FIRSTHDR(&msg);
		if (cmsg->cmsg_level == SOL_SOCKET && (cmsg->cmsg_type == SCM_RIGHTS))
			fd = *(int *)CMSG_DATA(cmsg);
	}
	return fd;
}

/* optv must be null-terminated */
int get_fuse_sock(connection_t *conn, int optc, char *const optv[])
{
	char **argv;
	char *envp[2];
	char *mount_notice, *arg_acc;
	pid_t fusermount_pid;
	int fuse_socks[2];
	int status;
	int fd;

	/* prepare argv from optv */
	argv = (char **)must_malloc("fusermount argv",
				    (optc + 2) * sizeof(char *));

	argv[0] = conn->params->fusermount;
	memcpy(&argv[1], optv, (optc + 1) * sizeof(char *));

	/* report the mount command issued */
	if (asprintf(&arg_acc, "mount") == -1)
		die(1, conn->params,
		    "Couldn't allocate mount notice base string", "");

	for (int i = 0; argv[i]; i++) {
		if (asprintf(&mount_notice, "%s %s", arg_acc, argv[i]) == -1)
			die(1, conn->params, "",
			    "Couldn't allocate mount notice arg %d: ", i);
		free(arg_acc);
		arg_acc = mount_notice;
	}

	if (asprintf(&mount_notice, "%s\n", arg_acc) == -1)
		die(1, conn->params, "Couldn't allocate mount notice", "");

	log_notice_time(conn->params, mount_notice);

	free(mount_notice);
	free(arg_acc);

	/* make the socket over which we'll be sent the FUSE socket fd */
	if (socketpair(PF_UNIX, SOCK_STREAM, 0, fuse_socks))
		die(1, conn->params, "Couldn't create FUSE socketpair", "");

	/* prepare to exec the suid binary fusermount */
	if (asprintf(&envp[0], "_FUSE_COMMFD=%d", fuse_socks[0]) == -1)
		die(1, conn->params, "Couldn't allocate fusermount envp", "");

	envp[1] = 0x0;

	/* fork and exec fusermount */
	fusermount_pid = fork();
	if (!fusermount_pid)
		/* child */
		if (execve(argv[0], argv, envp))
			die(1, conn->params,
			    "Failed to execute fusermount", "");

	/* parent */
	free(argv);
	free(envp[0]);

	/* close the end of the socket that we gave away */
	close(fuse_socks[0]);

	/* wait for fusermount to return */
	waitpid(fusermount_pid, &status, 0);
	if (!WIFEXITED(status))
		die(1, conn->params, NULL, "fusermount terminated abnormally");

	if (WEXITSTATUS(status))
		die(1, conn->params, NULL,
		    "fusermount exited with code %d", WEXITSTATUS(status));

	if (debug)
		log_time(conn->params, "about to recv_fd from fusermount\n");

	fd = recv_fd(conn->params, fuse_socks[1]);
	if (fd == -1)
		die(1, conn->params, NULL, "Couldn't receive fd over FUSE socket");

	/* close the read end of the socket */
	close(fuse_socks[1]);

	return fd;
}

void start_reader(connection_t *connection, int fuse)
{
	pthread_t child;
	copy_thread_state *copy_state;

	copy_state = (copy_thread_state *)
		must_malloc("start_reader copy_state",
			    sizeof(copy_thread_state));
	copy_state->connection = connection;
	copy_state->from = connection->sock;
	copy_state->to = fuse;
	errno = pthread_create(&child, &detached,
			       copy_clean_into_fuse_thread, copy_state);
	if (errno)
		die(1, connection->params, "",
		    "Couldn't create read copy thread for mount %s: ",
		    connection->mount_point);
}

void start_writer(connection_t *connection, int fuse)
{
	pthread_t child;
	copy_thread_state *copy_state;

	copy_state = (copy_thread_state *)
		must_malloc("start_writer copy_state",
			    sizeof(copy_thread_state));
	copy_state->connection = connection;
	copy_state->from = fuse;
	copy_state->to = connection->sock;
	errno = pthread_create(&child, &detached,
			       copy_clean_outof_fuse_thread, copy_state);
	if (errno)
		die(1, connection->params, "",
		    "Couldn't create write copy thread for mount %s: ",
		    connection->mount_point);
}

void negotiate_notify_channel(char *mount_point, int notify_sock)
{
	int len = strlen(mount_point);
	char hdr[6];

	*((uint32_t *)hdr) = 6 + len;
	*((uint16_t *)(hdr + 4)) = TRANSFUSE_NOTIFY_CHANNEL;

	write_exactly("negotiate_notify_channel hdr", notify_sock, hdr, 6);
	write_exactly("negotiate_notify_channel mnt",
		      notify_sock, mount_point, len);
}

void start_notify(connection_t *connection, int fuse)
{
	pthread_t child;
	copy_thread_state *copy_state;

	copy_state = (copy_thread_state *)
		must_malloc("start_notify copy_state",
			    sizeof(copy_thread_state));
	copy_state->connection = connection;
	copy_state->from = connect_socket(connection->params->server);
	copy_state->to = fuse;

	negotiate_notify_channel(connection->mount_point, copy_state->from);

	errno = pthread_create(&child, &detached,
			       copy_clean_notify_fuse_thread, copy_state);
	if (errno)
		die(1, connection->params, "",
		    "Couldn't create notify copy thread for mount %s: ",
		    connection->mount_point);
}


char *alloc_dirname(connection_t *conn, char *path)
{
	size_t len = strlen(path) + 1;
	char *input = must_malloc("alloc_dirname input", len);
	char *output = must_malloc("alloc_dirname output", len);
	char *dir;

	strlcpy(input, path, len);

	dir = dirname(input);
	if (dir == NULL)
		die(1, conn->params, "", "Couldn't get dirname of %s: ", path);
	strcpy(output, dir);

	free(input);
	return output;
}

void mkdir_p(connection_t *conn, char *path)
{
	char *parent;

	if (mkdir(path, 0700))
		switch (errno) {
		case EEXIST:
			return;
		case ENOENT:
			parent = alloc_dirname(conn, path);
			mkdir_p(conn, parent);
			free(parent);
			if (mkdir(path, 0700))
				die(1, conn->params, "",
				    "Couldn't create directory %s: ", path);
			break;
		default:
			die(1, conn->params, "",
			    "Couldn't create directory %s: ", path);
		}
}

int is_next_child_ok(parameters *params, char *path, DIR *dir)
{
	struct dirent *child;

	errno = 0;
	child = readdir(dir);
	if (child == NULL) {
		if (errno != 0)
			die(1, params, "",
			    "Couldn't read directory %s: ", path);
		else
			return 0;
	}
	return 1;
}

int is_path_mountable(parameters *params, int allow_empty, char *path)
{
	DIR *dir;

	dir = opendir(path);
	if (dir != NULL) {
		/* allow for . and .. */
		if (is_next_child_ok(params, path, dir))
			if (is_next_child_ok(params, path, dir)) {
				if (is_next_child_ok(params, path, dir))
					goto no;
				else if (allow_empty)
					goto yes;
				else
					goto no;
			}
		goto yes;
	} else {
		switch (errno) {
		case ENOENT:
			goto yes;
		case ENOTDIR:
			goto no;
		default:
			goto no;
		}
	}
	goto no;

no:
	if (dir)
		closedir(dir);
	return 0;

yes:
	if (dir)
		closedir(dir);
	return 1;
}

/* The leaf may exist but must be empty. Any proper path prefix may exist. */
void prepare_mount_point(connection_t *conn)
{
	char *mount_point = conn->mount_point;

	if (is_path_mountable(conn->params, 1, mount_point))
		mkdir_p(conn, mount_point);
	else
		die(1, conn->params, NULL,
		    "Couldn't mount on %s: not missing or empty", mount_point);
}

void *mount_connection(connection_t *conn)
{
	int optc;
	char **optv;
	int fuse;
	char *buf;
	pthread_mutex_t copy_lock;
	pthread_cond_t copy_halt;
	int should_halt = 0;

	buf = (char *)must_malloc("read_opts packet malloc", EVENT_BUFSZ);

	optv = read_opts(conn, buf);

	prepare_mount_point(conn);

	for (optc = 0; optv[optc] != NULL; optc++) {
	}

	fuse = get_fuse_sock(conn, optc, optv);
	free(buf);
	free(optv);

	lock_init("copy_lock", &copy_lock, NULL);
	cond_init("copy_halt", &copy_halt, NULL);

	start_reader(conn, fuse);
	start_writer(conn, fuse);
	start_notify(conn, fuse);

	lock("copy lock", &copy_lock);
	while (!should_halt)
		errno = pthread_cond_wait(&copy_halt, &copy_lock);
		if (errno)
			die(1, conn->params, "",
			    "Couldn't wait for copy halt for mount %s: ",
			    conn->mount_point);
	unlock("copy lock", &copy_lock);

	free(conn);

	return NULL;
}

void *mount_thread(void *connection)
{
	return mount_connection((connection_t *) connection);
}

void write_pid(connection_t *connection)
{
	pid_t pid = gettid();
	char *pid_s;
	int pid_s_len;

	if (asprintf(&pid_s, "%lld", (long long)pid) == -1)
		die(1, connection->params, "Couldn't allocate pid string", "");

	pid_s_len = strlen(pid_s);

	write_exactly("pid", connection->sock, pid_s, pid_s_len);

	free(pid_s);
}

void pong(parameters *params)
{
	char pong_msg[6] = {'\6', '\0', '\0', '\0', PONG_REPLY, '\0'};

	write_exactly("pong reply", params->ctl_sock, pong_msg, 6);
}

void perform_syscall(connection_t *conn, uint8_t syscall, char path[])
{
	char *name;
	int r = 0;

	switch (syscall) {

	case PING:
		pong(conn->params);
		r = 0;
		break;

	case RMDIR_SYSCALL:
		name = "rmdir";
		r = rmdir(path);
		break;

	case UNLINK_SYSCALL:
		name = "unlink";
		r = unlink(path);
		break;

	case MKDIR_SYSCALL:
		name = "mkdir";
		r = mkdir(path, 00000);
		break;

	case SYMLINK_SYSCALL:
		name = "symlink";
		r = symlink(".", path);
		break;

	case MKNOD_REG_SYSCALL:
		name = "mknod";
		r = mknod(path, 0600, 0);
		break;

	case TRUNCATE_SYSCALL:
		name = "truncate";
		r = truncate(path, 0);
		break;

	case CHMOD_SYSCALL:
		name = "chmod";
		r = chmod(path, 0700);
		break;

	default:
		die(1, conn->params, NULL,
		    "Unknown event syscall %" PRIu8, syscall);
	}

	if (r != 0)
		thread_log_time(conn, "Event %s %s error: %s\n",
				name, path, strerror(errno));
}

void *event_thread(void *connection_ptr)
{
	int read_count, path_len;
	void *buf;
	connection_t *connection = connection_ptr;

	char *path;
	uint8_t syscall;

	/* This thread registers with the file system server as being an
	 * fsnotify event actuator. Other mounted file system interactions
	 * (such as self-logging) SHOULD NOT occur on this thread. */
	write_pid(connection);

	buf = must_malloc("incoming event buffer", EVENT_BUFSZ);

	while (1) {
		read_count = read_message("events", connection->params,
					  connection->sock, buf, EVENT_BUFSZ);

		if (debug)
			thread_log_time(connection,
					"read %d bytes from event connection\n",
					read_count);

		path_len = (int)ntohs(*(((uint32_t *) buf) + 1));
		/* TODO: could check the path length isn't a lie here */
		path = (char *)(((uint8_t *)buf) + 6);
		/* TODO: could check the path is NULL terminated here */
		syscall = *(((uint8_t *)buf) + 6 + path_len);

		/* TODO: should this be in another thread ? */
		perform_syscall(connection, syscall, path);
	}

	free(buf);
	/* TODO: close connection */
	return NULL;
}

void write_pidfile(parameters *params)
{
	int fd;
	pid_t pid = getpid();
	char *pid_s;
	int pid_s_len, write_count;

	if (asprintf(&pid_s, "%lld", (long long)pid) == -1)
		die(1, params, "Couldn't allocate pidfile string", "");

	pid_s_len = strlen(pid_s);

	fd = open(params->pidfile, O_WRONLY | O_CREAT | O_TRUNC, 0644);
	if (fd == -1)
		die(1, params, "",
		    "Couldn't open pidfile path %s: ", params->pidfile);

	write_count = write(fd, pid_s, pid_s_len);
	if (write_count == -1)
		die(1, params, "",
		    "Error writing pidfile %s: ", params->pidfile);

	if (write_count != pid_s_len)
		die(1, params, NULL,
		    "Error writing %s to pidfile %s: only wrote %d bytes",
		    pid_s, params->pidfile, write_count);

	close(fd);
	free(pid_s);
}

/* TODO: the message parsing here is rickety, do it properly */
void *determine_mount_suitability(parameters *params, int allow_empty,
				  char *req, int len)
{
	void *buf = (void *)req;
	uint16_t id = *((uint16_t *) buf);
	uint16_t slen;
	char *reply;
	int roff;

	reply = (char *)must_malloc("determine_mount_suitability", len + 6);
	*((uint16_t *) (reply + 4)) = MOUNT_SUITABILITY_REPLY;
	*((uint16_t *) (reply + 6)) = id;
	roff = 8;

	buf = (void *)((char *)buf + 2);
	len -= 2;
	while (len) {
		slen = *((uint16_t *) buf) + 1;
		if (is_path_mountable(params, allow_empty, ((char *)buf) + 2)) {
			slen = strlcpy(reply + roff + 2,
				       ((char *)buf) + 2, slen) + 1;
			*((uint16_t *)((void *)(reply + roff))) = slen - 1;
			roff += 2 + slen;
		}
		buf = (void *)((char *)buf + 2 + slen);
		len -= 2 + slen;
	}

	*((uint32_t *) ((void *)reply)) = roff;
	return (void *)reply;
}

void *init_thread(void *params_ptr)
{
	parameters *params = params_ptr;
	int read_count, len;
	char init_msg[6] = {'\6', '\0', '\0', '\0', '\0', '\0'};
	void *buf, *response;
	uint16_t msg_type;

	params->ctl_sock = connect_socket(params->server);

	write_exactly("init", params->ctl_sock, init_msg, sizeof(init_msg));

	buf = must_malloc("incoming control message buffer", CTL_BUFSZ);

	/* TODO: handle other messages */
	read_exactly("init thread", params->ctl_sock, buf, 6);
	for (int i = 0; i < sizeof(init_msg); i++)
		if (((char *)buf)[i] != init_msg[i])
			die(1, params, NULL, "init thread: unexpected message");

	/* we've gotten Continue so write the pidfile */
	if (params->pidfile != NULL)
		write_pidfile(params);

	while (1) {
		read_count = read_message("control", params, params->ctl_sock,
					  buf, CTL_BUFSZ);
		msg_type = *((uint16_t *)buf + 2);
		switch (msg_type) {
		case MOUNT_SUITABILITY_REQUEST:
			response = determine_mount_suitability(params, 0,
							       (char *)buf + 6,
							       read_count - 6);
			len = *((size_t *) response);
			write_exactly("init thread: mount suitability response",
				      params->ctl_sock, response, len);
			free(response);
			break;

		case EXPORT_SUITABILITY_REQUEST:
			response = determine_mount_suitability(params, 1,
							       (char *)buf + 6,
							       read_count - 6);
			len = *((size_t *) response);
			write_exactly("init thread: export suitability response",
				      params->ctl_sock, response, len);
			free(response);
			break;

		default:
			die(1, params, NULL,
			    "init thread: unknown message %d", msg_type);
		}
	}

	free(buf);
	return NULL;
}

void toggle_debug(int sig)
{
	debug = !debug;
}

void setup_debug(void)
{
	if (signal(SIGHUP, toggle_debug) == SIG_ERR)
		die(1, NULL, "Couldn't set SIGHUP behavior", "");

	if (siginterrupt(SIGHUP, 1))
		die(1, NULL, "Couldn't set siginterrupt for SIGHUP", "");
}

void parse_parameters(int argc, char *argv[], parameters *params)
{
	int c;
	int errflg = 0;

	params->pidfile = NULL;
	params->socket = NULL;
	params->fusermount = NULL;
	params->logfile = NULL;
	params->logfile_fd = 0;
	params->data_sock = 0;
	params->ctl_sock = 0;
	lock_init("ctl_lock", &params->ctl_lock, NULL);

	while ((c = getopt(argc, argv, ":p:d:s:f:l:")) != -1) {
		switch (c) {

		case 'p':
			params->pidfile = optarg;
			break;

		case 'd':
			params->server = optarg;
			break;

		case 's':
			params->socket = optarg;
			break;

		case 'f':
			params->fusermount = optarg;
			break;

		case 'l':
			params->logfile = optarg;
			break;

		case ':':
			fprintf(stderr, "Option -%c requires a path argument\n", optopt);
			errflg++;
			break;

		case '?':
			fprintf(stderr, "Unrecognized option: '-%c'\n", optopt);
			errflg++;
			break;

		default:
			fprintf(stderr, "Internal error parsing -%c\n", c);
			errflg++;
		}
	}

	if (errflg) {
		fprintf(stderr, "%s", usage);
		exit(2);
	}
	if (params->pidfile != NULL && access(params->pidfile, W_OK))
		if (errno != ENOENT) {
			fprintf(stderr, "-p %s path to pidfile must be writable: ",
				params->pidfile);
			perror("");
			exit(2);
		}
	if (params->fusermount == NULL)
		params->fusermount = default_fusermount;
	if (access(params->fusermount, X_OK)) {
		fprintf(stderr, "-f %s path to fusermount must be executable: ",
			params->fusermount);
		perror("");
		exit(2);
	}
	if (params->socket == NULL)
		params->socket = default_socket;

	if (params->server == NULL)
		params->server = default_server;

	if (params->logfile != NULL && access(params->logfile, W_OK))
		if (errno != ENOENT) {
			fprintf(stderr, "-l %s path to logfile must be writable: ",
				params->logfile);
			perror("");
			exit(2);
		}
}

void serve(parameters *params)
{
	char subproto_selector;
	pthread_t child;
	connection_t *conn;
	void *(*connection_handler_thread)(void *);

	if (listen(params->data_sock, 16))
		die(1, NULL, "listen", "");

	errno = pthread_create(&child, &detached, init_thread, params);
	if (errno)
		die(1, NULL, "", "Couldn't create initialization thread: ");

	while (1) {
		conn = (connection_t *)must_malloc("connection state",
						   sizeof(connection_t));
		conn->params = params;
		conn->mount_point = "";

		conn->sock = accept(params->data_sock,
				    &conn->sa_client, &conn->socklen_client);
		if (conn->sock < 0)
			die(1, params, "accept", "");

		read_exactly("subproto", conn->sock, &subproto_selector, 1);

		switch (subproto_selector) {
		case 'm':
			conn->type_descr = "mount";
			connection_handler_thread = mount_thread;
			break;
		case 'e':
			conn->type_descr = "event";
			connection_handler_thread = event_thread;
			break;
		default:
			die(1, params, NULL,
			    "Unknown subprotocol type '%c'", subproto_selector);
		}

		errno = pthread_create(&child, &detached,
				       connection_handler_thread, conn);
		if (errno)
			die(1, params, "",
			    "Couldn't create thread for %s connection: ",
			    conn->type_descr);

		if (debug)
			log_time(conn->params, "thread spawned\n");
	}
}

int main(int argc, char *argv[])
{
	parameters params;
	struct rlimit core_limit;

	core_limit.rlim_cur = RLIM_INFINITY;
	core_limit.rlim_max = RLIM_INFINITY;
	if (setrlimit(RLIMIT_CORE, &core_limit))
		die(1, NULL, "", "Couldn't set RLIMIT_CORE to RLIM_INFINITY");

	openlog(argv[0], LOG_CONS | LOG_PERROR | LOG_NDELAY, LOG_DAEMON);

	parse_parameters(argc, argv, &params);
	setup_debug();

	errno = pthread_attr_setdetachstate(&detached, PTHREAD_CREATE_DETACHED);
	if (errno)
		die(1, NULL, "Couldn't set pthread detach state", "");

	if (params.logfile != NULL) {
		params.logfile_fd = open(params.logfile,
					 O_WRONLY | O_APPEND | O_CREAT);
		if (params.logfile_fd == -1)
			die(1, NULL, "",
			    "Couldn't open log file %s: ", params.logfile);
	}
	params.data_sock = bind_socket(params.socket);
	serve(&params);

	return 0;
}
