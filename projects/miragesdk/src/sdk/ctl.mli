(** [Control] handle the server part of the control path, running in
    the privileged container. *)


exception Undefined_field of int

module Endpoint: sig

  type t
  (** The type for SDK endpoints. *)

  val v: IO.t ->t
  (** [v f] is a fresh endpoint state built on top of the flow [f]. *)

  (** The type for endpoint errors. *)
  type error = private [>
    | `IO of IO.write_error
    | `Msg of string
    | `Undefined_field of int
  ]

  val pp_error: error Fmt.t
  (** [pp_error] is the pretty-printer for errors. *)

end

module Request: sig

  type t
  (** The type for SDK requests. *)

  (** The type for request actions. *)
  type action =
    | Write of string
    | Read
    | Delete

  val pp: t Fmt.t
  (** [pp] is the pretty-printer for requests. *)

  val equal: t -> t -> bool
  (** [equal] is the equality function for requests. *)

  val pp_action: action Fmt.t
  (** [pp_action] is the pretty-printer for request actions. *)

  val action: t -> action
  (** [action t] is [t]'s requested operation. Can raise
      [Endpoint.Undefined_field]. *)

  val path: t -> string list
  (** [path t] is the [t]'s request path. *)

  val id: t -> int32
  (** [id t] it [t]'s request id. *)

  val v: id:int32 -> path:string list -> action -> t
  (** [v ~id ~path action] is a new request. *)

  val write: Endpoint.t -> t -> (unit, Endpoint.error) result Lwt.t
  (** [write e t] writes a request message for the
      action [action] and the path [path] using the unique ID [id]. *)

  val read: Endpoint.t -> (t, Endpoint.error) result Lwt.t
  (** [read e] reads a query message. *)

end

module Response: sig

  type t
  (** The type for responses. *)

  (** The type for response status. *)
  type status = (string, string) result

  val pp: t Fmt.t
  (** [pp] is the pretty-printer for responses. *)

  val equal: t -> t -> bool
  (** [equal] is the equality function for responses. *)

  val pp_status: status Fmt.t
  (** [pp_status] is the pretty-printer for response statuses. *)

  val status: t -> status
  (** [status t] is [t]'s response status. Can raise
      [Endpoint.Undefined_field]. *)

  val id: t -> int32
  (** [id t] is [t]'s response ID. *)

  val v: id:int32 -> status -> t
  (** [v ~id status] is a new response. *)

  val write: Endpoint.t -> t -> (unit, Endpoint.error) result Lwt.t
  (** [write fd t] writes a reply message. *)

  val read: Endpoint.t -> (t, Endpoint.error) result Lwt.t
  (** [read fd] reads a reply message. *)

end

module Client: sig

  (** Client-side of the control plane. The control plane state is a
      simple KV store that the client can query with read/write/delete
      operations.

      TODO: decide if we want to support test_and_set (instead of
      write) and some kind of watches. *)

  type t
  (** The type for client state. *)

  type error
  (** The type for client errors. *)

  val pp_error: error Fmt.t
  (** [pp_error] is the pretty-printer for client errors. *)

  val v: IO.t -> t
  (** [v fd] is the client state using [fd] to send requests to the
      server. A client state also stores some state for all the
      incomplete client queries. *)

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

  val listen: routes:(string list * op list) list -> KV.t -> IO.t -> unit Lwt.t
  (** [listen ~routes kv fd] is the thread exposing the KV store [kv],
      holding control plane state, running inside the privileged
      container. [routes] are the routes exposed by the server to the
      calf and [kv] is the control plane state. *)

end
