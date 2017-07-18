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

module R = Api.Reader.Host
module B = Api.Builder.Host

module Client (F: Flow.S) = struct

  let pp_error = Capnp_rpc.Error.pp

  type t = R.t Capability.t

  let error e = Fmt.kstrf Lwt.fail_with "%a" pp_error e

  let connect ~switch ?tags f =
    let ep = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) f in
    let client = Capnp_rpc_lwt.CapTP.connect ~switch ?tags ep in
    Capnp_rpc_lwt.CapTP.bootstrap client |> Lwt.return

  let intf_result r =
    let module R = R.Intf_results in
    R.intf_get (R.of_payload r)

  let interface t =
    let module P = B.Intf_params in
    let req, _ = Capability.Request.create P.init_pointer in
    Capability.call_for_value t R.intf_method req >>= function
    | Error e -> error e
    | Ok r    -> Lwt.return (intf_result r)

  let mac_result r =
    let module R = R.Mac_results in
    let mac = R.mac_get (R.of_payload r) in
    Macaddr.of_string_exn mac

  let mac t =
    let module P = B.Mac_params in
    let req, _ = Capability.Request.create P.init_pointer in
    Capability.call_for_value t R.mac_method req >>= function
    | Error e -> error e
    | Ok r    -> Lwt.return (mac_result r)

  let dhcp_options_result r =
    let module R = R.DhcpOptions_results in
    let options = R.options_get_list (R.of_payload r) in
    List.fold_left (fun acc o ->
        match Dhcp_wire.string_to_option_code o with
        | None   -> acc
        | Some o -> o :: acc
      ) [] options

  let dhcp_options t =
    let module P = B.DhcpOptions_params in
    let req, _ = Capability.Request.create P.init_pointer in
    Capability.call_for_value t R.dhcp_options_method req >>= function
    | Error e -> error e
    | Ok r    -> Lwt.return (dhcp_options_result r)

  let set_ip t ip =
    let module P = B.SetIp_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.ip_set p (Ipaddr.V4.to_string ip);
    Capability.call_for_value t R.set_ip_method req >>= function
    | Error e -> error e
    | Ok _    -> Lwt.return ()

  let set_gateway t ip =
    let module P = B.SetGateway_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.ip_set p (Ipaddr.V4.to_string ip);
    Capability.call_for_value t R.set_gateway_method req >>= function
    | Error e -> error e
    | Ok _r   -> Lwt.return ()

end

module Server (F: Flow.S) (N: S) = struct

  type t = B.t Capability.t

  let mac_result result =
    let module R = B.Mac_results in
    let resp, r = Service.Response.create R.init_pointer in
    R.mac_set r (Macaddr.to_string result);
    Ok resp

  let intf_result result =
    let module R = B.Intf_results in
    let resp, r = Service.Response.create R.init_pointer in
    R.intf_set r result;
    Ok resp

  let dhcp_options_result result =
    let module R = B.DhcpOptions_results in
    let resp, r = Service.Response.create R.init_pointer in
    let result = List.map Dhcp_wire.option_code_to_string result in
    let _ = R.options_set_list r result in
    Ok resp

  let service t =
    B.local @@
    object (_ : B.service)
      inherit B.service

      method intf_impl _req =
        Service.return_lwt (fun () -> N.interface t >|= intf_result)

      method mac_impl _req =
        Service.return_lwt (fun () -> N.mac t >|= mac_result)

      method dhcp_options_impl _req =
        Service.return_lwt (fun () -> N.dhcp_options t >|= dhcp_options_result)

      method set_ip_impl req =
        let module P = R.SetIp_params in
        let params = P.of_payload req in
        let ip = P.ip_get params in
        Service.return_lwt (fun () ->
            let module R = B.SetIp_results in
            let resp, _ = Service.Response.create R.init_pointer in
            match Ipaddr.V4.of_string ip with
            | None    ->Lwt.fail_invalid_arg "invalid ip"
            | Some ip -> N.set_ip t ip >|= fun () -> Ok resp
          )

      method set_gateway_impl req =
        let module P = R.SetGateway_params in
        let params = P.of_payload req in
        let ip = P.ip_get params in
        Service.return_lwt (fun () ->
            let module R = B.SetGateway_results in
            let resp, _ = Service.Response.create R.init_pointer in
            match Ipaddr.V4.of_string ip with
            | None    ->Lwt.fail_invalid_arg "invalid ip"
            | Some ip -> N.set_ip t ip >|= fun () -> Ok resp
          )

    end

  let listen ~switch ?tags service fd =
    let endpoint = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) fd in
    Capnp_rpc_lwt.CapTP.connect ~switch ?tags ~offer:service endpoint
    |> ignore

end
