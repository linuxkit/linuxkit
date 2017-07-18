(** [Conf] exposes functions to to manipulate configuration data. *)

exception Undefined_field of int

module Client (F: Flow.S): sig

  (** [Client] exposes functions to read, write and watch
      configuration data. The configuration data is organized as a
      simple Irmin's KV store.

      {e TODO: decide if we want to support test_and_set (instead of
         write).} *)

  type t
  (** The type for client state. *)

  val connect: switch:Lwt_switch.t -> ?tags:Logs.Tag.set -> F.t -> t Lwt.t
  (** [connect f] connects to the flow [f]. *)

  val find: t -> string list -> string option Lwt.t
  (** [find t k] is the value associated with the key [k] in the
      control plane state. Return [None] if no value is associated to
      [k]. *)

  val get: t -> string list -> string Lwt.t
  (** [get t k] is similar to [fint t k] but raise `Invalid_argument`
      if the [k] is not a valid path. *)

  val set: t -> string list -> string -> unit Lwt.t
  (** [set t p v] associates [v] to the key [k] in the control plane
      state. *)

  val delete: t -> string list -> unit Lwt.t
  (** [delete t k] remove [k]'s binding in the control plane state. *)

  val watch: t -> string list -> (string -> unit Lwt.t) -> unit Lwt.t
  (** [watch t k f] calls [f] on every change of the key [k]. *)

end

module Server (F: Flow.S): sig

  (** [Server] exposes functions to serve configuration data over
      MirageOS flows. *)

  (** [KV] is the Irmin store storing configuration data. *)
  module KV: sig

    include Irmin.KV with type contents = string

    val v: unit -> t Lwt.t
    (** [v ()] is the KV store storing the control plane state. *)

  end

  type t
  (** The type for server state. *)

  type op = [ `Read | `Write | `Delete ]
  (** The type for operations to perform on routes. *)

  val service: switch:Lwt_switch.t ->
    routes:(string list * op list) list -> KV.t -> t
  (** [service ~switch ~routes kv] is the thread exposing the KV store [kv],
      holding control plane state, running inside the privileged
      container. [routes] are the routes exposed by the server to the
      calf and [kv] is the control plane state. *)

  val listen: switch:Lwt_switch.t -> ?tags:Logs.Tag.set -> t -> F.t -> unit
  (** [listen ~switch s m f] exposes service [s] on the flow
      [f]. [switch] can be used to stop the server. *)

end
