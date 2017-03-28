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

(* FIXME: this code is way too much complex *)
module HTTP (Net: Mirage_net_lwt.S) = struct
  module Flow = Raw(Net)
  module Channel = Mirage_channel_lwt.Make(Flow)
  (* FIXME: copy/pasted from mirage-http to avoid the dependency chain:
      mirage-http -> mirage-conduit -> nocrypto -> gmp -> .so needed  *)
  module HTTP_IO = struct
    type 'a t = 'a Lwt.t
    type ic = Channel.t
    type oc = Channel.t
    type conn = Channel.flow
    let failf fmt = Fmt.kstrf Lwt.fail_with fmt
    let read_line ic =
      Channel.read_line ic >>= function
      | Ok (`Data [])   -> Lwt.return_none
      | Ok `Eof         -> Lwt.return_none
      | Ok (`Data bufs) -> Lwt.return (Some (Cstruct.copyv bufs))
      | Error e         -> failf "Flow error: %a" Channel.pp_error e
    let read ic len =
      Channel.read_some ~len ic >>= function
      | Ok (`Data buf) -> Lwt.return (Cstruct.to_string buf)
      | Ok `Eof        -> Lwt.return ""
      | Error e        -> failf "Flow error: %a" Channel.pp_error e
    let write oc buf =
      Channel.write_string oc buf 0 (String.length buf);
      Channel.flush oc >>= function
      | Ok ()         -> Lwt.return_unit
      | Error `Closed -> Lwt.fail_with "Trying to write on closed channel"
      | Error e       -> failf "Flow error: %a" Channel.pp_write_error e
    let flush _ = Lwt.return_unit
    let  (>>= ) = Lwt.( >>= )
    let return = Lwt.return
  end
  module Net_IO = struct
    module IO = HTTP_IO
    type ctx = Net.t option
    let default_ctx = None
    let sexp_of_ctx _ = Sexplib.Sexp.Atom "netif"
    let connect_uri ~ctx _uri =
      match ctx with
      | None     -> Lwt.fail_with "No context"
      | Some ctx ->
        Flow.connect ctx >|= fun flow ->
        let ch = Channel.create flow in
        flow, ch, ch
    let close_in _ic = ()
    let close_out _oc = ()
    let close ic _oc = Lwt.ignore_result (Channel.close ic)
  end
  include Cohttp_lwt.Make_client(HTTP_IO)(Net_IO)
end

module API (Store: Mirage_net_lwt.S) = struct

  module HTTP = HTTP(Store)

  let http_post t uri ~body =
    HTTP.post ~ctx:(Some t) ~body:(`String body) uri >|= fun (response, _) ->
    (* FIXME check that response is ok *)
    Log.info
      (fun l -> l "POST %a: %a" Uri.pp_hum uri Cohttp.Response.pp_hum response)

  let set_ip t ip =
    http_post t (Uri.of_string "/ip") ~body:(Ipaddr.V4.to_string ip)

end


module Main
    (Time :Mirage_time_lwt.S)
    (Net  : Mirage_net_lwt.S)
    (Ctl  : Mirage_net_lwt.S) =
struct

  module API = API(Ctl)
  module Dhcp_client = Dhcp_client_mirage.Make(Time)(Net)

  let start () net ctl =
    let requests = match Key_gen.codes () with
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
    API.set_ip ctl result.address

end
