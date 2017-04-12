(** [Net] exposes low-level system functions related to network. *)

val mac: string -> Macaddr.t Lwt.t
(** [mac e] is the MAC address of the interface [e]. *)

val set_ip: string -> Ipaddr.V4.t -> unit Lwt.t
(** [set_ip e ip] sets [e]'s IP address to [ip]. *)

val set_gateway: Ipaddr.V4.t -> unit Lwt.t
(** [set_gateway ip] set the default host gateway to [ip]. *)
