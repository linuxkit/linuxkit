open Lwt.Infix
open Capnp_rpc_lwt

module B = Api.Builder.Net
module R = Api.Reader.Net

module type S = Mirage_net_lwt.S

module Callback = struct

  let service f =
    B.Callback.local @@ object (_: B.Callback.service)
      inherit B.Callback.service
      method f_impl req =
        let module P = R.Callback.F_params in
        let params = P.of_payload req in
        let change = P.buffer_get params in
        Service.return_lwt (fun () ->
            f (Cstruct.of_string change) >|= fun () ->
            Ok (Service.Response.create_empty ())
          )
    end

  module F = Api.Reader.Conf.Callback

  let client t change =
    let module P = B.Callback.F_params in
    let req, p = Capability.Request.create P.init_pointer in
    let change = Cstruct.to_string change in
    P.buffer_set p change;
    Capability.call_for_value t R.Callback.f_method req >>= function
    | Ok _    -> Lwt.return ()
    | Error e ->
      Fmt.kstrf Lwt.fail_with "error: f(%s) -> %a" change Capnp_rpc.Error.pp e

end

module Client (F: Flow.S) = struct

  type 'a io = 'a Lwt.t

  type t = {
    cap  : R.t Capability.t;
    mac  : Macaddr.t;
    stats: Mirage_net.stats;
  }

  type page_aligned_buffer = Io_page.t
  type buffer = Cstruct.t
  type macaddr = Macaddr.t

  type error = [
    | `Msg of string
    | `Undefined of int
    | `Capnp of Capnp_rpc.Error.t
    | Mirage_device.error
  ]

  let pp_error: error Fmt.t = fun ppf -> function
    | `Msg s       -> Fmt.pf ppf "error %s" s
    | `Undefined i -> Fmt.pf ppf "undefined %d" i
    | `Capnp e     -> Fmt.pf ppf "capnp: %a" Capnp_rpc.Error.pp e
    | #Mirage_device.error as e -> Mirage_device.pp_error ppf e

  let result r =
    let module R = R.Result in
    match R.get (R.of_payload r) with
    | R.Ok             -> Ok ()
    | R.Unimplemented -> Error `Unimplemented
    | R.Disconnected  -> Error `Disconnected
    | R.Error s       -> Error (`Msg s)
    | R.Undefined i   -> Error (`Undefined i)

  let write t buf =
    let module P = B.Write_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.buffer_set p (Cstruct.to_string buf);
    Capability.call_for_value t.cap R.write_method req >|= function
    | Error e -> Error (`Capnp e)
    | Ok r    ->
      Mirage_net.Stats.tx t.stats (Int64.of_int @@ Cstruct.len buf);
      result r

  let writev t bufs =
    let module P = B.Writev_params in
    let req, p = Capability.Request.create P.init_pointer in
    ignore @@ P.buffers_set_list p (List.map Cstruct.to_string bufs);
    Capability.call_for_value t.cap R.writev_method req >|= function
    | Error e -> Error (`Capnp e)
    | Ok r    ->
      Mirage_net.Stats.tx t.stats (Int64.of_int @@ Cstruct.lenv bufs);
      result r

  let listen t f =
    let module P = B.Listen_params in
    let req, p = Capability.Request.create P.init_pointer in
    let callback = Capability.Request.export req (Callback.service f) in
    P.callback_set p (Some callback);
    Capability.call_for_value t.cap R.listen_method req >|= function
    | Ok _    -> Ok ()
    | Error e -> Error (`Capnp e)

  let disconnect { cap; _ } =
    let module P = B.Disconnect_params in
    let req, _ = Capability.Request.create P.init_pointer in
    Capability.call_for_value cap R.disconnect_method req >|= fun _ ->
    ()

  let mac t = t.mac

  let capability ~switch ?tags f =
    let ep = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) f in
    let client = Capnp_rpc_lwt.CapTP.connect ~switch ?tags ep in
    Capnp_rpc_lwt.CapTP.bootstrap client |> Lwt.return

  let connect ~switch ?tags f =
    capability ~switch ?tags f >>= fun cap ->
    let module P = B.Mac_params in
    let req, _ = Capability.Request.create P.init_pointer in
    Capability.call_for_value cap R.mac_method req >>= function
    | Error e -> Fmt.kstrf Lwt.fail_with "%a" Capnp_rpc.Error.pp e
    | Ok r    ->
      let module R = R.Mac_results in
      let mac = R.mac_get (R.of_payload r) |> Macaddr.of_string_exn in
      let stats = Mirage_net.Stats.create () in
      Lwt.return { cap; mac; stats }

  let reset_stats_counters t = Mirage_net.Stats.reset t.stats
  let get_stats_counters t = t.stats
end

module Server (F: Flow.S) (Local: Mirage_net_lwt.S) = struct

  let result x =
    let module R = B.Result in
    let resp, r = Service.Response.create R.init_pointer in
    let () = match x with
      | Ok ()                -> R.ok_set r
      | Error `Disconnected  -> R.disconnected_set r
      | Error `Unimplemented -> R.unimplemented_set r
      | Error e              -> Fmt.kstrf (R.error_set r) "%a" Local.pp_error e
    in
    Ok resp

  let mac_result x =
    let module R = B.Mac_results in
    let resp, r = Service.Response.create R.init_pointer in
    R.mac_set r (Macaddr.to_string x);
    Ok resp

  let disconnect_result () =
    let module R = B.Disconnect_results in
    let resp, _ = Service.Response.create R.init_pointer in
    Ok resp

  let service t =
    B.local @@
    object (_ : B.service)
      inherit B.service

      method disconnect_impl _req =
        Service.return_lwt (fun () -> Local.disconnect t >|= disconnect_result)

      method write_impl req =
        let module P = R.Write_params in
        let params = P.of_payload req in
        let buf = P.buffer_get params |> Cstruct.of_string in
        Service.return_lwt (fun () -> Local.write t buf >|= result)

      method writev_impl req =
        let module P = R.Writev_params in
        let params = P.of_payload req in
        let bufs = P.buffers_get_list params |> List.map Cstruct.of_string in
        Service.return_lwt (fun () -> Local.writev t bufs >|= result)

      method listen_impl req =
        let module P = R.Listen_params in
        let params = P.of_payload req in
        match P.callback_get params with
        | None   -> failwith "No watcher callback given"
        | Some i ->
          let callback = Payload.import req i in
          Service.return_lwt (fun () ->
              Local.listen t (Callback.client callback) >|= result
            )

      method mac_impl req =
        let module P = R.Mac_params in
        let _params = P.of_payload req in
        Service.return_lwt (fun () -> Lwt.return (mac_result (Local.mac t)))

    end

  type t = R.t Capability.t

  let listen ~switch ?tags service fd =
    let endpoint = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) fd in
    Capnp_rpc_lwt.CapTP.connect ~switch ?tags ~offer:service endpoint
    |> ignore

end

let src = Logs.Src.create "sdk/net"
module Log = (val Logs.src_log src : Logs.LOG)

module Fd = struct

  module Net = Mirage_net_flow.Make(Flow.Fd)

  include Net

  let connect ?mac (i:int) =
    let fd : Unix.file_descr = Obj.magic i in
    let fd = Lwt_unix.of_unix_file_descr fd in
    Net.connect ?mac (Flow.Fd.of_fd fd)

end

module Rawlink = struct

  module R = Mirage_flow_rawlink
  module Net = Mirage_net_flow.Make(R)
  include Net

  let connect ~filter ?mac ethif =
    Log.debug (fun l -> l "bringing up %s" ethif);
    (try Tuntap.set_up_and_running ethif
     with e -> Log.err (fun l -> l "rawlink: %a" Fmt.exn e));
    let flow = Lwt_rawlink.open_link ~filter ethif in
    connect ?mac flow

end
