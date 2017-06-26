open Lwt.Infix

let src = Logs.Src.create "dhcp-client/engine"
module Log = (val Logs.src_log src : Logs.LOG)

type t = {
  address: Ipaddr.V4.t;
  gateway: Ipaddr.V4.t option;
  domain: string option;
  search: string option;
  nameservers: Ipaddr.V4.t list;
}

(* FIXME: we (still) lose lots of info here *)
let of_lease (t: Dhcp_wire.pkt) =
  let gateway = match Dhcp_wire.collect_routers t.Dhcp_wire.options with
  | [] -> None
  | n::_ -> Some n
  in
  { address = t.Dhcp_wire.yiaddr;
    gateway;
    domain = Dhcp_wire.find_domain_name t.Dhcp_wire.options;
    search = Dhcp_wire.find_domain_search t.Dhcp_wire.options;
    nameservers = Dhcp_wire.collect_dns_servers t.Dhcp_wire.options }

let pp ppf t =
  Fmt.pf ppf "\n\
              address    : %a\n\
              domain     : %a\n\
              search     : %a\n\
              nameservers: %a\n"
    Ipaddr.V4.pp_hum t.address
    Fmt.(option ~none:(unit "--") string) t.domain
    Fmt.(option ~none:(unit "--") string) t.search
    Fmt.(list ~sep:(unit " ") Ipaddr.V4.pp_hum) t.nameservers

module Make
    (Time: Sdk.Time.S)
    (Net : Sdk.Net.S)
    (Host: Sdk.Host.S) =
struct

  module Dhcp_client = Dhcp_client_lwt.Make(Time)(Net)

  let start _ net host =
    Host.dhcp_options host >>= fun requests ->
    Dhcp_client.connect ~requests net >>= fun stream ->
    Lwt_stream.last_new stream >>= fun result ->
    let result = of_lease result in
    Log.info (fun l -> l "found lease: %a" pp result);
    Host.set_ip host result.address >>= fun () ->
    (match result.gateway with
     | None -> Lwt.return_unit
     | Some ip -> Host.set_gateway host ip)

end
