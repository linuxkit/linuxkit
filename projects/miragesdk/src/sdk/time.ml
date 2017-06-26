module type S = sig
  type t
  include Mirage_time_lwt.S
end

module Local = struct
  type +'a io = 'a Lwt.t
  type t = unit
  let sleep_ns x = Lwt_unix.sleep (Int64.to_float x /. 1_000_000_000.)
end
