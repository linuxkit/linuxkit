val bind: string -> Lwt_unix.file_descr Lwt.t
val connect: string -> Lwt_unix.file_descr Lwt.t
val send_fd: to_send:Lwt_unix.file_descr -> Lwt_unix.file_descr -> unit Lwt.t
val recv_fd: Lwt_unix.file_descr -> Lwt_unix.file_descr Lwt.t
