(** [Control] handle the server part of the control path, running in
    the privileged container. *)


exception Undefined_field of int

module Client: sig

  (** Client-side of the control plane. The control plane state is a
      simple KV store that the client can query with read/write/delete
      operations.

      TODO: decide if we want to support test_and_set (instead of
      write) and some kind of watches. *)

  type t = Ctl_api.Reader.Ctl.t Capnp_rpc_lwt.Capability.t
  (** The type for client state. *)

  type error
  (** The type for client errors. *)

  val pp_error: error Fmt.t
  (** [pp_error] is the pretty-printer for client errors. *)

  val read: t -> string list -> (string option, error) result Lwt.t
  (** [read t k] is the value associated with the key [k] in the
      control plane state. Return [None] if no value is associated to
      [k]. *)

  val write: t -> string list -> string -> (unit, error) result Lwt.t
  (** [write t p v] associates [v] to the key [k] in the control plane
      state. *)

  val delete: t -> string list -> (unit, error) result Lwt.t
  (** [delete t k] remove [k]'s binding in the control plane state. *)

end

(** [KV] stores tje control plane state. *)
module KV: Irmin.KV with type contents = string

val v: string -> KV.t Lwt.t
(** [v p] is the KV store storing the control plane state, located at
    path [p] in the filesystem of the privileged container. *)

module Server: sig

  type op = [ `Read | `Write | `Delete ]
  (** The type for operations to perform on routes. *)

  val service: routes:(string list * op list) list -> KV.t -> Ctl_api.Reader.Ctl.t Capnp_rpc_lwt.Capability.t
  (** [service ~routes kv] is the thread exposing the KV store [kv],
      holding control plane state, running inside the privileged
      container. [routes] are the routes exposed by the server to the
      calf and [kv] is the control plane state. *)

end
