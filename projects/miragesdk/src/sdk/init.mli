(** Init functions.

    [Init] contains funcitons to initialise the state of the
    privileged container.

    {ul

    {- fowrard and filter the network traffic using BPF (for instance
       to allow only DHCP traffic).}
    {- open pipes for forwarding the calf's stdout and stderr
       to the privileged container's ones.}
    {- open a pipe to forward the metrics.}
    {- open a socket pair with the calf to be able to transmit control
       data, e.g. the IP address once a DHCP lease is obtained.}
    }*)


module Fd: sig

  type t
  (** The type for file descriptors. *)

  val pp: t Fmt.t
  (** [pp_fd] pretty prints a file descriptor. *)

  val redirect_to_dev_null: t -> unit
  (** [redirect_to_dev_null fd] redirects [fd] [/dev/null]. *)

  val close: t -> unit
  (** [close fd] closes [fd]. *)

  val dup2: src:t -> dst:t -> unit
  (** [dup2 ~src ~dst] calls [Unix.dup2] on [src] and [dst]. *)

  (** {1 Usefull File Descriptors} *)

  val stdin: t
  (** [stdin] is the standart input. *)

  val stdout: t
  (** [stdout] is the standard output. *)

  val stderr: t
  (** [stderr] is the standard error. *)

  val flow: t -> IO.t
  (** [flow t] is the flow representing [t]. *)

end

val file_descr: ?name:string -> Lwt_unix.file_descr -> IO.t
(** [file_descr ?name fd] is the flow for the file-descripor [fd]. *)

module Pipe: sig

  type t
  (** The type for pipes. Could be either uni-directional (normal
      pipes) or a bi-directional (socket pairs). *)

  type monitor
  (** The type for pipe monitors. *)

  val v: unit -> monitor

  val name: t -> string
  (** [name t] is [t]'s name. *)

  val priv: t -> Fd.t
  (** [priv p] is the private side of the pipe [p]. *)

  val calf: t -> Fd.t
  (** [calf p] is the calf side of the pipe [p]. *)

  (** {1 Useful Pipes} *)

  val stdout: monitor -> t
  (** [stdout m] is the uni-directional pipe from the calf's stdout
      monitored by [m]. *)

  val stderr: monitor -> t
  (** [stderr m] is the uni-directional pipe from the calf's stderr
      monitored by [m]. *)

  val metrics: monitor -> t
  (** [metrics m] is the uni-directional pipe from the calf's metric
      endpoint monitored by [m]. *)

  val ctl: monitor -> t
  (** [ctl m] is the bi-directional pipe used to exchange control data
      between the calf and the priv containers monitored by [m]. *)

  val net: monitor -> t
  (** [net m] is the bi-directional pipe used to exchange network
      traffic between the calf and the priv containers monitored by
      [m]. *)

end

val rawlink: ?filter:string -> string -> IO.t
(** [rawlink ?filter x] is the flow using the network interface
    [x]. The packets can be filtered using the BPF filter
    [filter]. See the documentation of
    {{:https://github.com/haesbaert/rawlink}rawlink} for more details
    on how to build that filter. *)

val exec: Pipe.monitor -> string list -> (int -> unit Lwt.t) -> unit Lwt.t
(** [exec t cmd k] executes [cmd] in an unprivileged calf process and
    call [k] with the pid of the parent process. The child and parents
    are connected using [t]. *)

(* FIXME(samoht): not very happy with that signatue *)
val run: Pipe.monitor ->
  net:IO.t -> ctl:(IO.t -> unit Lwt.t) ->
  ?handlers:(unit -> unit Lwt.t) ->
  string list -> unit Lwt.t
(** [run m ~net ~ctl ?handlers cmd] runs [cmd] in a unprivileged calf
    process. [net] is the network interface flow. [ctl] is the control
    thread connected to the {Pipe.ctl} pipe. [handlers] are the system
    handler thread which will react to control data to perform
    privileged system actions. *)
