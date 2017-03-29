(** [Control] handle the server part of the control path, running in
    the privileged container. *)

module KV: Irmin.KV with type contents = string

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
