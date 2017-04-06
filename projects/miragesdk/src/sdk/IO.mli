(** IO helpers *)

type t
(** The type for IO flows *)

include Mirage_flow_lwt.S with type flow = t

val create: (module Mirage_flow_lwt.S with type flow = 'a) -> 'a -> string -> flow
(** [create (module M) t name] is the flow representing [t] using the
    function defined in [M]. *)

val pp: flow Fmt.t
(** [pp] is the pretty-printer for IO flows. *)

val forward: src:t -> dst:t -> unit Lwt.t
(** [forward ~src ~dst] forwards writes from [src] to [dst]. Block
    until either [src] or [dst] is closed. *)

val proxy: t -> t -> unit Lwt.t
(** [proxy x y] is the same as [forward x y <*> forward y x]. Block
    until both flows are closed. *)
