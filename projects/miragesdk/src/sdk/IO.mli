(** IO helpers *)

val really_write: Lwt_unix.file_descr -> string -> int -> int -> unit Lwt.t
(** [really_write fd buf off len] writes exactly [len] bytes to [fd]. *)

val really_read: Lwt_unix.file_descr -> string -> int -> int -> unit Lwt.t
(** [really_read fd buf off len] reads exactly [len] bytes from [fd]. *)

val read_all: Lwt_unix.file_descr -> string Lwt.t
(** [read_all fd] reads as much data as it is available in [fd]. *)

val read_n: Lwt_unix.file_descr -> int -> string Lwt.t
(** [read_n fd n] reads exactly [n] bytes from [fd]. *)
