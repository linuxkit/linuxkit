open Lwt.Infix

let src = Logs.Src.create "charrua"
module Log = (val Logs.src_log src : Logs.LOG)

type t = {
  address: Ipaddr.V4.t;
  domain: string option;
  search: string option;
  nameservers: Ipaddr.V4.t list;
}

(* FIXME: we loose lots of info here *)
let of_ipv4_config (t: Mirage_protocols_lwt.ipv4_config) =
  { address = t.Mirage_protocols_lwt.address;
    domain = None;
    search = None;
    nameservers = [] }

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

let of_pkt lease =
  let open Dhcp_wire in
  (* ipv4_config expects a single IP address and the information
   * needed to construct a prefix. It can optionally use one router. *)
  let address = lease.yiaddr in
  let domain = Dhcp_wire.find_domain_name lease.options in
  let search = Dhcp_wire.find_domain_search lease.options in
  let nameservers = Dhcp_wire.collect_name_servers lease.options in
  { address; domain; search; nameservers }

let of_pkt_opt = function
  | None       -> None
  | Some lease -> Some (of_pkt lease)

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

(* Build a raw flow from a network interface *)
module Raw (Net: Mirage_net_lwt.S): sig
  include Mirage_flow_lwt.S
  val connect: Net.t -> flow Lwt.t
end = struct

  type 'a io = 'a Net.io
  type error = Net.error
  let pp_error = Net.pp_error
  type write_error = [ Mirage_flow.write_error | `Net of Net.error ]

  let pp_write_error ppf = function
    | #Mirage_flow.write_error as e -> Mirage_flow.pp_write_error ppf e
    | `Net e -> Net.pp_error ppf e

  type flow = {
    netif: Net.t;
    mutable closed: bool;
    listener: unit Lwt.t;
    bufs: Cstruct.t Queue.t;
    cond: [`Eof | `Data] Lwt_condition.t;
  }

  type buffer = Cstruct.t

  let connect netif =
    let cond = Lwt_condition.create () in
    let bufs = Queue.create () in
    let listener =
      Net.listen netif (fun buf ->
          Queue.add buf bufs;
          Lwt_condition.signal cond `Data;
          Lwt.return_unit)
      >|= function
      | Ok ()   -> ()
      | Error e ->
        Log.debug (fun l -> l "net->flow listen: %a" Net.pp_error e);
        Lwt_condition.broadcast cond `Eof
    in
    Lwt.return { netif; bufs; cond; closed = false; listener }

  let read flow =
    if flow.closed then Lwt.return (Error `Disconnected)
    else if Queue.is_empty flow.bufs then
      Lwt_condition.wait flow.cond >|= function
      | `Eof  -> Ok `Eof
      | `Data -> Ok (`Data (Queue.pop flow.bufs))
    else
      Lwt.return (Ok (`Data (Queue.pop flow.bufs)))

  let close flow =
    flow.closed <- true;
    Lwt.cancel flow.listener;
    Lwt.return_unit

  let writev t bufs =
    if t.closed then Lwt.return (Error `Closed)
    else Net.writev t.netif bufs >|= function
      | Ok ()   -> Ok ()
      | Error e -> Error (`Net e)

  let write t buf =
    if t.closed then Lwt.return (Error `Closed)
    else Net.write t.netif buf >|= function
      | Ok ()   -> Ok ()
      | Error e -> Error (`Net e)

end

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

module Dhcp_client = Dhcp_client_mirage.Make(Time)(Net)

let start () dhcp_codes net ctl =
  Netif_fd.connect net >>= fun net ->
  let ctl = Sdk.Ctl.Client.v (Lwt_unix.of_unix_file_descr ctl) in
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
  let result = of_ipv4_config result in
  Log.info (fun l -> l "found lease: %a" pp result);
  Sdk.Ctl.Client.write ctl "/ip" (Ipaddr.V4.to_string result.address ^ "\n")

(* FIXME: Main end *)
let magic (x: int) = (Obj.magic x: Unix.file_descr)

let start () dhcp_codes net ctl =
  Lwt_main.run (
    let net = magic net in
    let ctl = magic ctl in
    start () dhcp_codes net ctl
  )

let run =
  Term.(const start $ setup_log $ dhcp_codes $ net $ ctl),
  Term.info "dhcp-client" ~version:"0.0"

let () = match Term.eval run with
  | `Error _ -> exit 1
  | `Ok (Ok ()) |`Help |`Version -> exit 0
  | `Ok (Error (`Msg e)) ->
    Printf.eprintf "%s\n%!" e;
    exit 1
