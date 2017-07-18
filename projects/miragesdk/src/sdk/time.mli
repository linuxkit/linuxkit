(** MirageOS's time interface over RPC. *)

module type S = sig
  type t
  include Mirage_time_lwt.S
end

module Local: S with type t = unit
