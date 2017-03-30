(** [Control] handle the server part of the control path, running in
    the privileged container. *)

module KV: Irmin.KV with type contents = string

module Message: sig

  (** The type for operations. *)
  type operation =
    | Write
    | Read
    | Delete

  (** The type for control messages. *)
  type t = {
    operation: operation;
    path     : string;
    payload  : string option;
  }

  val write_message: Lwt_unix.file_descr -> t -> unit Lwt.t
  (** [write_message fd t] writes a control message. *)

  val read_message: Lwt_unix.file_descr -> t Lwt.t
  (** [read_message fd] reads a control message. *)

end

val v: string -> KV.t Lwt.t
(** [v p] is the KV store storing the control state, located at path
    [p] in the filesystem of the privileged container. *)

val serve: routes:string list -> KV.t -> Lwt_unix.file_descr -> unit Lwt.t
(** [serve ~routes kv fd] is the thread exposing the KV store [kv],
    holding control state, running inside the privileged container.
    [routes] are the routes exposed by the server (currently over a
    simple HTTP server -- but will change to something else later,
    probably protobuf) to the calf and [kv] is the control state
    handler. *)
