(** [Engine] is the DHCP client engine. It access network traffic via
    the [Net] MirageOS's network interface, and use [Act] to modify IP
    tables and other low-level caches. *)

module Make
    (Time: Sdk.Time.S)
    (Net : Sdk.Net.S)
    (Act : Sdk.Host.S):
sig
  val start: Time.t -> Net.t -> Act.t -> unit Lwt.t
end
