(** [Control] handle the server part of the control path, running in
    the privileged container. *)

module Query: sig

  (** The type for operations. *)
  type operation =
    | Write
    | Read
    | Delete

  (** The type for control plane queries. *)
  type t = {
    version  : int32;                                   (** Protocol version. *)
    id       : int32;                                 (** Session identifier. *)
    operation: operation;
    path     : string;                        (** Should be only valid ASCII. *)
    payload  : string;                                 (** Arbitrary payload. *)
  }

  type error = [ `Eof | `Msg of string ]
  (** The type of errors. *)

  val pp_error: error Fmt.t
  (** [pp_error] is the pretty-printer for query errors. *)

  val pp: t Fmt.t
  (** [pp] is the pretty-printer for queries. *)

  val of_cstruct: Cstruct.t -> (t, [`Msg of string]) result
  (** [of_cstruct buf] is the query [t] such that the serialization of
      [t] is [buf]. *)

  val to_cstruct: t -> Cstruct.t
  (** [to_cstruct t] is the serialization of [t]. *)

  val write: IO.flow -> t -> unit Lwt.t
  (** [write fd t] writes a query message. *)

  val read: IO.flow -> (t, error) result Lwt.t
  (** [read fd] reads a query message. *)

end

module Reply: sig

  (** The type for status. *)
  type status =
    | Ok
    | Error

  (** The type for control plane replies. *)
  type t = {
    id     : int32;                                   (** Session identifier. *)
    status : status;                             (** Status of the operation. *)
    payload: string;                                   (** Arbitrary payload. *)
  }

  val pp: t Fmt.t
  (** [pp] is the pretty-printer for replies. *)

  val of_cstruct: Cstruct.t -> (t, [`Msg of string]) result
  (** [of_cstruct buf] is the reply [t] such that the serialization of
      [t] is [buf]. *)

  val to_cstruct: t -> Cstruct.t
  (** [to_cstruct t] is the serialization of [t]. *)

  type error = [`Eof | `Msg of string ]
  (** The type for reply errors. *)

  val pp_error: error Fmt.t
  (** [pp_error] is the pretty-printer for errors. *)

  val write: IO.flow -> t -> unit Lwt.t
  (** [write fd t] writes a reply message. *)

  val read: IO.flow -> (t, error) result Lwt.t
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

  val read: t -> string -> (string option, error) result Lwt.t
  (** [read t k] is the value associated with the key [k] in the
      control plane state. Return [None] if no value is associated to
      [k]. *)

  val write: t -> string -> string -> (unit, error) result Lwt.t
  (** [write t p v] associates [v] to the key [k] in the control plane
      state. *)

  val delete: t -> string -> (unit, error) result Lwt.t
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

  val listen: routes:(string * op list) list -> KV.t -> IO.t -> unit Lwt.t
  (** [listen ~routes kv fd] is the thread exposing the KV store [kv],
      holding control plane state, running inside the privileged
      container. [routes] are the routes exposed by the server to the
      calf and [kv] is the control plane state. *)

end
