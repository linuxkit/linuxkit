open Lwt.Infix
open Capnp_rpc_lwt

module type S = Mirage_net_lwt.S

module Callback = struct

  let service f =
    let open Api.Service.Net.Callback in
    local @@ object (_: service)
      inherit service
      method f_impl req release_param_caps =
        let change = F.Params.buffer_get req in
        release_param_caps ();
        Service.return_lwt (fun () ->
            f (Cstruct.of_string change) >|= fun () ->
            Ok (Service.Response.create_empty ())
          )
    end

  let client t change =
    let open Api.Client.Net.Callback in
    let req, p = Capability.Request.create F.Params.init_pointer in
    let change = Cstruct.to_string change in
    F.Params.buffer_set p change;
    Capability.call_for_value_exn t F.method_id req >|= ignore

end

module Client (F: Flow.S) = struct

  module Net = Api.Client.Net

  type 'a io = 'a Lwt.t

  type t = {
    cap  : Net.t Capability.t;
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

  let result r: (unit, error) result =
    let module R = Net.Write.Results in
    match R.get r with
    | R.Ok            -> Ok ()
    | R.Unimplemented -> Error `Unimplemented
    | R.Disconnected  -> Error `Disconnected
    | R.Error s       -> Error (`Msg s)
    | R.Undefined i   -> Error (`Undefined i)

  let write t buf =
    let open Net.Write in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.buffer_set p (Cstruct.to_string buf);
    Capability.call_for_value t.cap method_id req >|= function
    | Error e -> Error (`Capnp e)
    | Ok r    ->
      Mirage_net.Stats.tx t.stats (Int64.of_int @@ Cstruct.len buf);
      result r

  let writev t bufs =
    let open Net.Writev in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.buffers_set_list p (List.map Cstruct.to_string bufs) |> ignore;
    Capability.call_for_value t.cap method_id req >|= function
    | Error e -> Error (`Capnp e)
    | Ok r    ->
      Mirage_net.Stats.tx t.stats (Int64.of_int @@ Cstruct.lenv bufs);
      result r

  let listen t f =
    let open Net.Listen in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.callback_set p (Some (Callback.service f));
    Capability.call_for_value t.cap method_id req >|= function
    | Ok _    -> Ok ()
    | Error e -> Error (`Capnp e)

  let disconnect { cap; _ } =
    let open Net.Disconnect in
    let req, _ = Capability.Request.create Params.init_pointer in
    Capability.call_for_value_exn cap method_id req >|=
    ignore

  let mac t = t.mac

  let capability ~switch ?tags f =
    let ep = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) f in
    let client = Capnp_rpc_lwt.CapTP.connect ~switch ?tags ep in
    Capnp_rpc_lwt.CapTP.bootstrap client |> Lwt.return

  let connect ~switch ?tags f =
    let open Net.Mac in
    capability ~switch ?tags f >>= fun cap ->
    let req, _ = Capability.Request.create Params.init_pointer in
    Capability.call_for_value_exn cap method_id req >|= fun r ->
    let mac = Results.mac_get r |> Macaddr.of_string_exn in
    let stats = Mirage_net.Stats.create () in
    { cap; mac; stats }

  let reset_stats_counters t = Mirage_net.Stats.reset t.stats
  let get_stats_counters t = t.stats
end

module Server (F: Flow.S) (Local: Mirage_net_lwt.S) = struct

  module Net = Api.Service.Net

  let result x =
    let module R = Net.Write.Results in
    let resp, r = Service.Response.create R.init_pointer in
    let () = match x with
      | Ok ()                -> R.ok_set r
      | Error `Disconnected  -> R.disconnected_set r
      | Error `Unimplemented -> R.unimplemented_set r
      | Error e              -> Fmt.kstrf (R.error_set r) "%a" Local.pp_error e
    in
    Ok resp

  let mac_result x =
    let module R = Net.Mac.Results in
    let resp, r = Service.Response.create R.init_pointer in
    R.mac_set r (Macaddr.to_string x);
    resp

  let disconnect_result () =
    let module R = Net.Disconnect.Results in
    let resp, _ = Service.Response.create R.init_pointer in
    Ok resp

  let service t =
    Net.local @@ object (_ : Net.service)
      inherit Net.service

      method disconnect_impl _req release_param_caps =
        release_param_caps ();
        Service.return_lwt (fun () -> Local.disconnect t >|= disconnect_result)

      method write_impl req release_param_caps =
        let open Net.Write in
        let buf = Params.buffer_get req |> Cstruct.of_string in
        release_param_caps ();
        Service.return_lwt (fun () -> Local.write t buf >|= result)

      method writev_impl req release_param_caps =
        let open Net.Writev in
        let bufs = Params.buffers_get_list req |> List.map Cstruct.of_string in
        release_param_caps ();
        Service.return_lwt (fun () -> Local.writev t bufs >|= result)

      method listen_impl req release_param_caps =
        let open Net.Listen in
        let callback = Params.callback_get req in
        release_param_caps ();
        match callback with
        | None   -> Service.fail "No watcher callback given"
        | Some i ->
          Service.return_lwt (fun () ->
              Local.listen t (Callback.client i) >|= result
            )

      method mac_impl _req release_param_caps =
        release_param_caps ();
        Service.return (mac_result (Local.mac t))

    end

  type t = Net.t Capability.t

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
