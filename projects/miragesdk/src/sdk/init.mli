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

  val fd: t -> Lwt_unix.file_descr
  (** [fd t] is [t]'s underlying unix file descriptor. *)

  val to_int: t -> int
(** [to_int fd] is [fd]'s number. *)

  val redirect_to_dev_null: t -> unit Lwt.t
  (** [redirect_to_dev_null fd] redirects [fd] [/dev/null]. *)

  val close: t -> unit Lwt.t
  (** [close fd] closes [fd]. *)

  val dup2: src:t -> dst:t -> unit Lwt.t
  (** [dup2 ~src ~dst] calls [Unix.dup2] on [src] and [dst]. *)

  val proxy_net: net:Lwt_rawlink.t -> t -> unit Lwt.t
  (** [proxy_net ~net fd] proxies the traffic between the raw net link
      [net] and [fd]. *)

  val forward: src:t -> dst:t -> unit Lwt.t
  (** [forward ~src ~dst] forwards the flow from [src] to [dst]. *)

  (** {1 Usefull File Descriptors} *)

  val stdin: t
  (** [stdin] is the standart input. *)

  val stdout: t
  (** [stdout] is the standard output. *)

  val stderr: t
  (** [stderr] is the standard error. *)

end

module Pipe: sig

  type t
  (** The type for pipes. Could be either uni-directional (normal
      pipes) or a bi-directional (socket pairs). *)

  val priv: t -> Fd.t
  (** [priv p] is the private side of the pipe [p]. *)

  val calf: t -> Fd.t
  (** [calf p] is the calf side of the pipe [p]. *)

  (** {1 Useful Pipes} *)

  val stdout: t
  (** [stdout] is the uni-directional pipe from the calf's stdout . *)

  val stderr: t
  (** [stderr] is the uni-directional pipe from the calf's stderr. *)

  val metrics: t
  (** [metrics] is the uni-directional pipe fomr the calf's metric
      endpoint. *)

  val ctl: t
  (** [ctl] is the bi-directional pipe used to exchange control
      data between the calf and the priv containers. *)

  val net: t
  (** [net] is the bi-directional pipe used to exchange network
      traffic between the calf and the priv containers. *)

end

val rawlink: ?filter:string -> string -> Lwt_rawlink.t
(** [rawlink ?filter i] is the net raw link to the interface [i] using
    the (optional) BPF filter [filter]. *)

val run:
  net:Lwt_rawlink.t ->
  ctl:(unit -> unit Lwt.t) ->
  handlers:(unit -> unit Lwt.t) ->
  string list -> unit Lwt.t
(** [run ~net ~ctl ~handlers cmd] runs [cmd] in a unprivileged calf
    process. [ctl] is the control thread connected to the {Pipe.ctl}
    pipe. [net] is the net raw link which will be connected to the
    calf via the {!Pipe.net} socket pair. [handlers] are the system
    handler thread which will react to control data to perform
    privileged system actions. *)
