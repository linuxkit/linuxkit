open Lwt.Infix

let src = Logs.Src.create "charrua"
module Log = (val Logs.src_log src : Logs.LOG)

let failf fmt = Fmt.kstrf Lwt.fail_with fmt

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

let parse_option_code str =
  match Dhcp_wire.string_to_option_code str with
  | Some x -> Ok x
  | None   -> Error (Fmt.strf "%s is not a valid DHCP option code" str)

let default_options =
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

(* FIXME: use the mirage tool *)

module Time = struct
  type +'a io = 'a Lwt.t
  let sleep_ns x = Lwt_unix.sleep (Int64.to_float x /. 1_000_000_000.)
end
module Net = Netif_fd
module Ctl = Netif_fd

open Cmdliner

let dhcp_codes =
  let doc = Arg.info ~docv:"OPT" ~doc:"DHCP options." ["c";"codes"] in
  Arg.(value & opt (list string) [] doc)

let net =
  let doc = Arg.info ~docv:"FD" ~doc:"Network interface" ["net"] in
  Arg.(value & opt int 3 doc)

let ctl =
  let doc = Arg.info ~docv:"FD" ~doc:"Control interface" ["ctl"] in
  Arg.(value & opt int 4 doc)

let setup_log style_renderer level =
  Fmt_tty.setup_std_outputs ?style_renderer ();
  Logs.set_level level;
  let pp_header ppf x =
    Fmt.pf ppf "%5d: %a " (Unix.getpid ()) Logs_fmt.pp_header x
  in
  Logs.set_reporter (Logs_fmt.reporter ~pp_header ());
  ()

let setup_log =
  Term.(const setup_log $ Fmt_cli.style_renderer () $ Logs_cli.level ())

(* FIXME: module Main ... *)

module Dhcp_client = Dhcp_client_lwt.Make(Time)(Net)

let pp_path = Fmt.(list ~sep:(unit "/") string)

let set_ip ctl k ip =
  let str = Ipaddr.V4.to_string ip ^ "\n" in
  Sdk.Ctl.Client.write ctl k str >>= function
  | Ok ()   -> Lwt.return_unit
  | Error e ->
    failf "error while writing %a: %a" pp_path k  Sdk.Ctl.Client.pp_error e

let set_ip_opt ctl k = function
  | None    -> Lwt.return_unit
  | Some ip -> set_ip ctl k ip

let get_mac ctl =
  Sdk.Ctl.Client.read ctl ["mac"] >>= function
  | Ok None   -> Lwt.return None
  | Ok Some s -> Lwt.return @@ Macaddr.of_string (String.trim s)
  | Error e   -> failf "get_mac: %a" Sdk.Ctl.Client.pp_error e

let start () dhcp_codes net ctl =
  get_mac ctl >>= fun mac ->
  Netif_fd.connect ?mac net >>= fun net ->
  let requests = match dhcp_codes with
    | [] -> default_options
    | l  ->
      List.fold_left (fun acc c -> match parse_option_code c with
          | Ok x    -> x :: acc
          | Error e ->
            Log.err (fun l -> l "error: %s" e);
            acc
        ) [] l
  in
  Dhcp_client.connect ~requests net >>= fun stream ->
  Lwt_stream.last_new stream >>= fun result ->
  let result = of_lease result in
  Log.info (fun l -> l "found lease: %a" pp result);
  set_ip ctl ["ip"] result.address >>= fun () ->
  set_ip_opt ctl ["gateway"] result.gateway

(* FIXME: Main end *)

let fd (x: int) = (Obj.magic x: Unix.file_descr)

let flow (x: int) = Sdk.Init.file_descr (Lwt_unix.of_unix_file_descr @@ fd x)

let start () dhcp_codes net ctl =
  Lwt_main.run (
    Lwt_switch.with_switch @@ fun switch ->
    let net = fd net in
    let flow =
      Capnp_rpc_lwt.Endpoint.of_flow ~switch (module Mirage_flow_lwt) (flow ctl)
    in
    let client = Capnp_rpc_lwt.CapTP.of_endpoint ~switch flow in
    let ctl = Capnp_rpc_lwt.CapTP.bootstrap client in
    start () dhcp_codes net ctl
  )

let run =
  Term.(const start $ setup_log $ dhcp_codes $ net $ ctl),
  Term.info "dhcp-client" ~version:"0.0"

let () = match Term.eval run with
  | `Error _ -> exit 1
  | `Ok () |`Help |`Version -> exit 0
