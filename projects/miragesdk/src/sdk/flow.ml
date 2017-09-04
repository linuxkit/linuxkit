open Lwt.Infix
open Capnp_rpc_lwt

module type S = sig
  type t
  include Mirage_flow_lwt.S with type flow = t
end

module Client (F: S) = struct

  module Flow = Api.Client.Flow

  type 'a io = 'a Lwt.t
  type t = Flow.t Capability.t
  type flow = t

  type buffer = Cstruct.t

  type error = [
    | `Msg of string
    | `Undefined of int
    | `Capnp of Capnp_rpc.Error.t
  ]

  type write_error = [
    | `Closed
    | error
  ]

  let pp_error: error Fmt.t = fun ppf -> function
    | `Msg s       -> Fmt.pf ppf "error %s" s
    | `Undefined i -> Fmt.pf ppf "undefined %d" i
    | `Capnp e     -> Fmt.pf ppf "capnp: %a" Capnp_rpc.Error.pp e

  let pp_write_error: write_error Fmt.t = fun ppf -> function
    | `Closed     -> Fmt.string ppf "closed"
    | #error as e -> pp_error ppf e

  let connect ~switch ?tags f =
    let ep = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) f in
    let client = Capnp_rpc_lwt.CapTP.connect ~switch ?tags ep in
    Capnp_rpc_lwt.CapTP.bootstrap client |> Lwt.return

  let read_result r =
    let module R = Flow.Read.Results in
    match R.get r with
    | R.Data data   -> Ok (`Data (Cstruct.of_string data))
    | R.Eof         -> Ok `Eof
    | R.Error s     -> Error  (`Msg s)
    | R.Undefined i -> Error (`Undefined i)

  let write_result r =
    let module R = Flow.Write.Results in
    match R.get r with
    | R.Ok          -> Ok ()
    | R.Closed      -> Error `Closed
    | R.Error s     -> Error (`Msg s)
    | R.Undefined i -> Error (`Undefined i)

  let read t =
    let open Flow.Read in
    let req, _ = Capability.Request.create Params.init_pointer in
    Capability.call_for_value t method_id req >|= function
    | Error e -> Error (`Capnp e)
    | Ok r    -> read_result r

  let write t buf =
    let open Flow.Write in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.buffer_set p (Cstruct.to_string buf);
    Capability.call_for_value t method_id req >|= function
    | Error e -> Error (`Capnp e)
    | Ok r    -> write_result r

  let writev t bufs =
    let open Flow.Writev in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.buffers_set_list p (List.map Cstruct.to_string bufs) |> ignore;
    Capability.call_for_value t method_id req >|= function
    | Error e -> Error (`Capnp e)
    | Ok r    -> write_result r

  let close t =
    let open Flow.Close in
    let req, _ = Capability.Request.create Params.init_pointer in
    Capability.call_for_value_exn t method_id req >|= ignore

end

module Server (F: S) (Local: S) = struct

  module Flow = Api.Service.Flow

  let read_result result =
    let module R = Flow.Read.Results in
    let resp, r = Service.Response.create R.init_pointer in
    let () = match result with
      | Ok (`Data buf) -> R.data_set r (Cstruct.to_string buf)
      | Ok `Eof        -> R.eof_set r
      | Error e        -> Fmt.kstrf (R.error_set r) "%a" Local.pp_error e
    in
    Ok resp

  let write_result result =
    let module R = Flow.Write.Results in
    let resp, r = Service.Response.create R.init_pointer in
    let () = match result with
      | Ok ()         -> R.ok_set r
      | Error `Closed -> R.closed_set r
      | Error e       -> Fmt.kstrf (R.error_set r) "%a" Local.pp_write_error e
    in
    Ok resp

  let close_result () =
    let module R = Flow.Close.Results in
    let resp, _ = Service.Response.create R.init_pointer in
    Ok resp

  let service t =
    Flow.local @@ object (_ : Flow.service)
      inherit Flow.service

      method read_impl _req release_param_caps =
        release_param_caps ();
        Service.return_lwt (fun () -> Local.read t >|= read_result)

      method write_impl req release_param_caps =
        let open Flow.Write in
        let buf = Params.buffer_get req |> Cstruct.of_string in
        release_param_caps ();
        Service.return_lwt (fun () -> Local.write t buf >|= write_result)

      method writev_impl req release_param_caps =
        let open Flow.Writev in
        let bufs = Params.buffers_get_list req |> List.map Cstruct.of_string in
        release_param_caps ();
        Service.return_lwt (fun () -> Local.writev t bufs >|= write_result)

      method close_impl _req release_param_caps =
        release_param_caps ();
        Service.return_lwt (fun () -> Local.close t >|= close_result)

    end

  type t = Flow.t Capability.t

  let listen ~switch ?tags service fd =
    let endpoint = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) fd in
    Capnp_rpc_lwt.CapTP.connect ~switch ?tags ~offer:service endpoint
    |> ignore

end

let src = Logs.Src.create "sdk/flow"
module Log = (val Logs.src_log src : Logs.LOG)

module FIFO = struct

  include Mirage_flow_unix.Fd
  type t = flow

  let mkfifo path =
    if not (Sys.file_exists path) then
      Lwt.catch (fun () ->
          Lwt_unix.mkfifo path 0o644
        ) (function
          | Unix.Unix_error(Unix.EEXIST, _, _) -> Lwt.return_unit
          | e -> Lwt.fail e)
    else
      Lwt.return_unit

  let of_fd x = x

  let connect path =
    Log.debug (fun l -> l "opening FIFO: %s\n%!" path);
    mkfifo path >>= fun () ->
    Lwt_unix.openfile path [Lwt_unix.O_RDWR] 0o644

end

module Socket = struct

  include Mirage_flow_unix.Fd
  type t = flow

  let connect path =
    let fd = Lwt_unix.socket Lwt_unix.PF_UNIX Lwt_unix.SOCK_STREAM 0 in
    Lwt_unix.connect fd (Lwt_unix.ADDR_UNIX path) >|= fun () ->
    fd

end

module Rawlink = struct

  include Mirage_flow_rawlink
  type t = flow

  let connect ~filter ethif =
    Log.debug (fun l -> l "bringing up %s" ethif);
    (try Tuntap.set_up_and_running ethif
     with e -> Log.err (fun l -> l "rawlink: %a" Fmt.exn e));
    Lwt_rawlink.open_link ~filter ethif
    |> Lwt.return

end

module Fd = struct

  include Mirage_flow_unix.Fd
  type t = flow

  let of_fd x = x

  let connect (i:int) =
    let fd : Unix.file_descr = Obj.magic i in
    Lwt_unix.of_unix_file_descr fd
    |> Lwt.return

end

module Mem = struct
  include Mirage_flow_lwt.F
  type t = flow
  let connect () = make ()
end
