(** [Network] provides a MirageOS's network interface with only DHCP
    traffic. It uses [Act] to get the host's MAC address. *)

module Make (Act: Sdk.Host.S): sig
  include Sdk.Net.S
  val connect: Act.t -> t Lwt.t
end
