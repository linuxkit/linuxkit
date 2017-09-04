(* This file is a big hack and should be replaced ASAP with proper bindings *)

open Lwt.Infix

let src = Logs.Src.create "net" ~doc:"Network Configuration"
module Log = (val Logs.src_log src : Logs.LOG)

module type S = sig
  type t
  val interface: t -> string Lwt.t
  val mac: t -> Macaddr.t Lwt.t
  val dhcp_options: t -> Dhcp_wire.option_code list Lwt.t
  val set_ip: t -> Ipaddr.V4.t -> unit Lwt.t
  val set_gateway: t -> Ipaddr.V4.t -> unit Lwt.t
end

module Local = struct

  type t = {
    intf: string
  }

  let connect intf = Lwt.return {intf}
  let interface {intf} = Lwt.return intf

  let run fmt =
    Fmt.kstrf (fun str ->
        Log.info (fun l -> l "run: %S" str);
        match Sys.command str with
        | 0 -> Lwt.return ()
        | i -> Fmt.kstrf Lwt.fail_with "%S exited with code %d" str i
      ) fmt

  let read fmt =
    Fmt.kstrf (fun str ->
        Lwt_process.pread ("/bin/sh", [|"/bin/sh"; "-c";  str|])
      ) fmt

  let mac t =
    read "ifconfig -a %s | grep -o -E '([[:xdigit:]]{1,2}:){5}[[:xdigit:]]{1,2}'"
      t.intf >|= fun mac ->
    Macaddr.of_string_exn (String.trim mac)

  let dhcp_options _t =
    (* FIXME: read /etc/dhcpc.conf *)
    let open Dhcp_wire in
    [
      RAPID_COMMIT;
      DOMAIN_NAME;
      DOMAIN_SEARCH;
      HOSTNAME;
      CLASSLESS_STATIC_ROUTE;
      NTP_SERVERS;
      INTERFACE_MTU;
    ]
    |> Lwt.return

  let set_ip t ip =
    (* FIXME: use language bindings to netlink instead *)
    (* run "ip addr add %s/24 dev %s" ip ethif *)
    run "ifconfig %s %a netmask 255.255.255.0" t.intf Ipaddr.V4.pp_hum ip

  let set_gateway _t gw =
    run "ip route add default via %a" Ipaddr.V4.pp_hum gw

end

open Lwt.Infix
open Capnp_rpc_lwt

module Client (F: Flow.S) = struct

  module Host = Api.Client.Host

  type t = Host.t Capability.t

  let connect ~switch ?tags f =
    let ep = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) f in
    let client = Capnp_rpc_lwt.CapTP.connect ~switch ?tags ep in
    Capnp_rpc_lwt.CapTP.bootstrap client |> Lwt.return

  let interface t =
    let open Host.Intf in
    let req, _ = Capability.Request.create Params.init_pointer in
    Capability.call_for_value_exn t method_id req >|=
    Host.Intf.Results.intf_get

  let mac t =
    let open Host.Mac in
    let req, _ = Capability.Request.create Params.init_pointer in
    Capability.call_for_value_exn t method_id req >|= fun r ->
    Macaddr.of_string_exn (Results.mac_get r)

  let dhcp_options t =
    let open Host.DhcpOptions in
    let req, _ = Capability.Request.create Params.init_pointer in
    Capability.call_for_value_exn t method_id req >|= fun r ->
    let options = Results.options_get_list r in
    List.fold_left (fun acc o ->
        match Dhcp_wire.string_to_option_code o with
        | None   -> acc
        | Some o -> o :: acc
      ) [] options

  let set_ip t ip =
    let open Host.SetIp in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.ip_set p (Ipaddr.V4.to_string ip);
    Capability.call_for_value_exn t method_id req >|=
    ignore

  let set_gateway t ip =
    let open Host.SetGateway in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.ip_set p (Ipaddr.V4.to_string ip);
    Capability.call_for_value_exn t method_id req >|=
    ignore

end

module Server (F: Flow.S) (N: S) = struct

  module Host = Api.Service.Host

  type t = Host.t Capability.t

  let mac_result result =
    let module R = Host.Mac.Results in
    let resp, r = Service.Response.create R.init_pointer in
    R.mac_set r (Macaddr.to_string result);
    Ok resp

  let intf_result result =
    let module R = Host.Intf.Results in
    let resp, r = Service.Response.create R.init_pointer in
    R.intf_set r result;
    Ok resp

  let dhcp_options_result result =
    let module R = Host.DhcpOptions.Results in
    let resp, r = Service.Response.create R.init_pointer in
    let result = List.map Dhcp_wire.option_code_to_string result in
    let _ = R.options_set_list r result in
    Ok resp

  let service t =
    Host.local @@ object (_ : Host.service)
      inherit Host.service

      method intf_impl _req release_param_caps =
        release_param_caps ();
        Service.return_lwt (fun () -> N.interface t >|= intf_result)

      method mac_impl _req release_param_caps =
        release_param_caps ();
        Service.return_lwt (fun () -> N.mac t >|= mac_result)

      method dhcp_options_impl _req release_param_caps =
        release_param_caps ();
        Service.return_lwt (fun () -> N.dhcp_options t >|= dhcp_options_result)

      method set_ip_impl req release_param_caps =
        let open Host.SetIp in
        let ip = Params.ip_get req in
        release_param_caps ();
        Service.return_lwt (fun () ->
            let resp, _ = Service.Response.create Results.init_pointer in
            match Ipaddr.V4.of_string ip with
            | None    -> Lwt.fail_with "invalid ip"
            | Some ip -> N.set_ip t ip >|= fun () -> Ok resp
          )

      method set_gateway_impl req release_param_caps =
        let open Host.SetGateway in
        let ip = Params.ip_get req in
        release_param_caps ();
        Service.return_lwt (fun () ->
            let resp, _ = Service.Response.create Results.init_pointer in
            match Ipaddr.V4.of_string ip with
            | None    -> Lwt.fail_invalid_arg "invalid ip"
            | Some ip -> N.set_gateway t ip >|= fun () -> Ok resp
          )

    end

  let listen ~switch ?tags service fd =
    let endpoint = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) fd in
    Capnp_rpc_lwt.CapTP.connect ~switch ?tags ~offer:service endpoint
    |> ignore

end
