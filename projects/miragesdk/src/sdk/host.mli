(** [Net] exposes low-level system functions related to network. *)

module type S = sig

  type t
  (** The type for network host actuator. *)

  val interface: t -> string Lwt.t
  (** [interface t] is [t]'s interface. *)

  val mac: t -> Macaddr.t Lwt.t
  (** [mac t] is the MAC address of the interface [e]. *)

  val dhcp_options: t -> Dhcp_wire.option_code list Lwt.t
  (** [dhcp_options] are the DHCP client options associted with the
      [t]'s interface. *)

  val set_ip: t -> Ipaddr.V4.t -> unit Lwt.t
  (** [set_ip t ip] sets [t]'s IP address to [ip]. *)

  val set_gateway: t -> Ipaddr.V4.t -> unit Lwt.t
  (** [set_gateway ip] set the default host gateway to [ip]. *)

end

(** [Client(F)] a an implementation of S interface over the flow
    [F]. Once connected, to the other side of the flow, behave just as
    a normal local net actuator, althought all the calls are now sent
    to the remote end. *)
module Client (F: Flow.S): sig
  include S
  val connect: switch:Lwt_switch.t -> ?tags:Logs.Tag.set -> F.t -> t Lwt.t
end

(** [Server(F)(Local)] exposes the host networking actuator [Local] as
    a Cap-n-p RPC endpoint over the flow [F]. Clients calls executed
    on the other end of the flow [F] will be executed on the
    server-side. *)
module Server (F: Flow.S) (Local: S): sig
  type t
  val service: Local.t -> t
  val listen: switch:Lwt_switch.t -> ?tags:Logs.Tag.set -> t -> F.t -> unit
end

(** Local network actuactor. At the moment uses a lot of very bad
    hacks, should be cleaned up. *)
module Local: sig
  include S
  val connect: string -> t Lwt.t
end
