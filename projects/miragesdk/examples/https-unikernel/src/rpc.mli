(** Cap'n Proto RPC adaptors.
    This module deals with sending and receiving RPC messages.
    It provides a more idiomatic OCaml API than the generated stubs.
    There is one sub-module for each RPC interface.
    Each sub-module provides client-side methods to invoke the
    service, and a [local] function to implement a service. *)

open Capnp_rpc_lwt

[@@@ocaml.warning "-34"]
(* See: https://caml.inria.fr/mantis/print_bug_page.php?bug_id=7438 *)

module Flow : sig
  include Mirage_flow_lwt.S with
    type flow = [`Flow_e102f5fcaceb1e06] Capability.t and
    type error = [`Capnp of Capnp_rpc.Error.t] and
    type write_error = [`Closed | `Capnp of Capnp_rpc.Error.t]

  val local : Lwt_io.input Lwt_io.channel -> Lwt_io.output Lwt_io.channel -> flow
  (** [local ic oc] is a capability to a local flow implemented by [ic] and [oc]. *)
end

module Store : sig
  type t = [`Store_96a6b45508236c12] Capability.t

  val get : t -> string list -> string option Lwt.t
  (** [get t path] looks up [path] in store [t]. *)

  val local : (string list -> string option Lwt.t) -> t
  (** [local lookup] is a local store that responds to requests with [lookup key]. *)
end

module Http : sig
  type t = [`HttpServer_9ecd1f7bbfef9f1e] Capability.t

  val accept : t -> Flow.flow -> unit Lwt.t
  (** [accept t flow] asks [t] to handle new connection [flow]. *)

  val local : (Flow.flow -> unit Lwt.t) -> t
  (** [local handle_connection] is a capability to a local HTTP service that
      uses [handle_connection flow] to handle each connection. *)
end
