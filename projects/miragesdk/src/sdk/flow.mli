(** MirageOS's flow interface over RPC *)

module type S = sig
  type t
  include Mirage_flow_lwt.S with type flow = t
end

(** {1 Remote Flows} *)

(** [Client(F)] a an implementation of MirageOS's flow interface over
    the flow [F]. Once connected, to the other side of the flow,
    behave just as a normal local flow, althought all the calls are
    now sent to the remote end. *)
module Client (F: S): sig
  include S
  val connect: switch:Lwt_switch.t -> ?tags:Logs.Tag.set -> F.t -> t Lwt.t
end

(** [Server(F)(Local)] exposes the flow [Local] as a Cap-n-p RPC
    endpoint over the flow [F]. Clients calls done on the other side
    of the flow [F] will be executed on the server-side. *)
module Server (F: S) (Local: S): sig
  type t
  val service: Local.t -> t
  val listen: switch:Lwt_switch.t -> ?tags:Logs.Tag.set -> t -> F.t -> unit
end

(** {1 Local Flows} *)

module FIFO: sig
  include S
  val of_fd: Lwt_unix.file_descr -> t
  val connect: string -> t Lwt.t
end

module Socket: sig
  include S
  val connect: string -> t Lwt.t
end

module Rawlink: sig
  include S
  val connect: filter:string -> string -> t Lwt.t
end

module Fd: sig
  include S
  val of_fd: Lwt_unix.file_descr -> t
  val connect: int -> t Lwt.t
end

module Mem: sig
  include S
  val connect: unit -> t
end
