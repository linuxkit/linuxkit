(** MirageOS's net interface over RPC *)

(** {1 Remote Networks} *)

module type S = Mirage_net_lwt.S

(** [Client(F)] a an implementation of MirageOS's net interface over
    the flow [F]. Once connected, to the other side of the net, behave
    just as a normal local net, althought all the calls are now sent
    to the remote end. *)
module Client (F: Flow.S): sig
  include S
  val connect: switch:Lwt_switch.t -> ?tags:Logs.Tag.set -> F.t -> t Lwt.t
end

(** [Server(F)(Local)] exposes the MirageOS's network [Local] as a
    Cap-n-p RPC endpoint over the flow [F]. Clients calls done on the
    other end of [F] will be executed on the server-side. *)
module Server (F: Flow.S) (Local: S): sig
  type t
  val service: Local.t -> t
  val listen: switch:Lwt_switch.t -> ?tags:Logs.Tag.set -> t -> F.t -> unit
end

(** {1 Local Networks} *)

module Fd: sig
  include S
  val connect: ?mac:Macaddr.t -> int -> t Lwt.t
end

module Rawlink: sig
  include S
  val connect: filter:string -> ?mac:Macaddr.t -> string -> t Lwt.t
end
