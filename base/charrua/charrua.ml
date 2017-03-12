open Lwt.Infix

let src = Logs.Src.create "charrua"
module Log = (val Logs.src_log src : Logs.LOG)

type t = {
  address: Ipaddr.V4.t;
  domain: string option;
  search: string option;
  nameservers: Ipaddr.V4.t list;
}

type json = [
  | `String of string
  | `A of json list
  | `O of (string * json) list
]

let json_to_dst ~minify dst (json:json) =
  let enc e l = ignore (Jsonm.encode e (`Lexeme l)) in
  let rec value v k e = match v with
  | `A vs -> arr vs k e
  | `O ms -> obj ms k e
  | `String _ as v -> enc e v; k e
  and arr vs k e = enc e `As; arr_vs vs k e
  and arr_vs vs k e = match vs with
  | v :: vs' -> value v (arr_vs vs' k) e
  | [] -> enc e `Ae; k e
  and obj ms k e = enc e `Os; obj_ms ms k e
  and obj_ms ms k e = match ms with
  | (n, v) :: ms -> enc e (`Name n); value v (obj_ms ms k) e
  | [] -> enc e `Oe; k e
  in
  let e = Jsonm.encoder ~minify dst in
  let finish e = ignore (Jsonm.encode e `End) in
  value json finish e

let to_json t =
  let string_opt k v = match v with
    | None   -> []
    | Some v -> [k, `String v]
  in
  let list k v = match v with
    | [] -> []
    | v  -> [k, `A (List.map (fun v -> `String v) v)]
  in
  let ip = Ipaddr.V4.to_string in
  let kv k v = [k, `String v] in
  `O (kv         "address"      (ip t.address) @
      string_opt "domain"       t.domain @
      string_opt "search"       t.search @
      list       "nameservers" (List.map ip t.nameservers))

let pp_json ppf t =
  let buf = Buffer.create 42 in
  json_to_dst (`Buffer buf) ~minify:false (to_json t);
  Fmt.string ppf (Buffer.contents buf)

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

module Net = Netif_fd

module Time = struct
  type +'a io = 'a Lwt.t
  let sleep_ns x = Lwt_unix.sleep (Int64.to_float x /. 1_000_000_000.)
end

let create ?requests net =
  (* listener needs to occasionally check to see whether the state has
   * advanced, and if not, start a new attempt at a lease
     transaction *)
  let sleep_interval = Duration.of_sec 5 in

  let (client, dhcpdiscover) = Dhcp_client.create ?requests (Net.mac net) in
  let c = ref client in

  let rec repeater dhcpdiscover =
    Log.debug (fun f -> f "Sending DHCPDISCOVER...");
    Net.write net dhcpdiscover >|= Rresult.R.get_ok >>= fun () ->
    Time.sleep_ns sleep_interval >>= fun () ->
    match of_pkt_opt (Dhcp_client.lease !c) with
    | Some lease ->
      Log.info (fun f -> f "Lease obtained! IP:\n%a" pp_json lease);
      Lwt.return (Some lease)
    | None ->
      let (client, dhcpdiscover) = Dhcp_client.create ?requests (Net.mac net) in
      c := client;
      Log.info (fun f -> f "Timeout expired without a usable lease!\
                            Starting over...");
      Log.debug (fun f -> f "New lease attempt: %a" Dhcp_client.pp !c);
      repeater dhcpdiscover
  in
  let listen () =
    Net.listen net (fun buf ->
        match Dhcp_client.input !c buf with
        | (s, Some action) ->
          Net.write net action >|=
          Rresult.R.get_ok >|= fun () ->
          Log.debug (fun f -> f "State advanced! Now %a" Dhcp_client.pp s);
          c := s
        | (s, None) ->
          Log.debug (fun f -> f "No action! State is %a" Dhcp_client.pp s);
          c := s;
          Lwt.return_unit
      ) >|= Rresult.R.get_ok
  in
  let get_lease () =
    Lwt.pick [ (listen () >|= fun () -> None);
               repeater dhcpdiscover; ]
  in
  Lwt_stream.from get_lease

let fd_of_int (x: int) : Unix.file_descr = Obj.magic x

let fd_of_path path =
  match Astring.String.cut ~sep:"fd:" path with
  | Some ("fd:", n) ->
    (try fd_of_int (int_of_string n)
     with Failure _ ->
       Log.err (fun e -> e "%s is not a valid path" path);
       exit 1)
  | _ ->
    let fd = Unix.socket Unix.PF_UNIX Unix.SOCK_RAW 0 in
    Unix.bind fd (Unix.ADDR_UNIX path);
    fd

let get requests input =
  Net.connect (fd_of_path input) >>= fun net ->
  Lwt_stream.last_new (create ~requests net)

let process () requests input output =
  Lwt_main.run (
    get requests input >>= fun x ->
    let buf = Cstruct.of_string (Fmt.to_to_string pp_json x) in
    let output = fd_of_path output in
    let output = Lwt_unix.of_unix_file_descr output in
    Lwt_cstruct.(complete (write output) buf)
  )

open Cmdliner

let stdin = "fd:0"
let stdout = "fd:1"

let input =
  let doc =
    Arg.info ~docv:"PATH"
      ~doc:"Unix domain socket to get input from. Use fd:<id> to use an \
            already open file-descriptor. If not provided, use stdin"
      ["input"; "i"]
  in
  Arg.(value & opt string stdin & doc)

let output =
  let doc =
    Arg.info ~docv:"PATH"
      ~doc:"Unix domain socket to publish results to. Use fd:<id> to use an \
            already open file-descriptor. If not provided, use stdout."
      ["output"; "o"]
  in
  Arg.(value & opt string stdout & doc)

let option_code =
  let parse str =
    match Dhcp_wire.string_to_option_code str with
    | Some x -> `Ok x
    | None   -> `Error (Fmt.strf "%s is not a valid DHCP option code" str)
  in
  let pp ppf t = Fmt.string ppf (Dhcp_wire.option_code_to_string t) in
  parse, pp

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

let dhcp_codes =
  let doc = Arg.info ~docv:"OPT" ~doc:"DHCP options." ["c";"codes"] in
  Arg.(value & opt (list option_code) default_options & doc)

let pp_ptime f () =
  let open Unix in
  let tm = Unix.localtime (Unix.time ()) in
  Fmt.pf f "%04d-%02d-%02d %02d:%02d"
    (tm.tm_year + 1900) (tm.tm_mon + 1) tm.tm_mday tm.tm_hour tm.tm_min

let reporter =
  let report src level ~over k msgf =
    let k _ = over (); k () in
    let ppf = Fmt.stderr in
    let with_stamp h _tags k fmt =
      Fmt.kpf k ppf ("\r%a %a %a @[" ^^ fmt ^^ "@]@.")
        pp_ptime ()
        Fmt.(styled `Magenta string) (Printf.sprintf "%10s" @@ Logs.Src.name src)
        Logs_fmt.pp_header (level, h)
    in
    msgf @@ fun ?header ?tags fmt ->
    with_stamp header tags k fmt
  in
  { Logs.report = report }

let setup_log =
  let env =
    Arg.env_var ~docs:"LOG OPTIONS"
      ~doc:"Be more or less verbose. See $(b,--verbose)."
      "DATAKIT_VERBOSE"
  in
  let f style_renderer level =
    Logs.set_level level;
    Fmt_tty.setup_std_outputs ?style_renderer ();
    Logs.set_reporter reporter
  in
  Term.(const f $ Fmt_cli.style_renderer () $ Logs_cli.level ~env ())

let term =
  let doc = "DHCP client" in
  let man = [
    `S "DESCRIPTION";
    `P "Simple DHCP client."
  ] in
  Term.(pure process $ setup_log $ dhcp_codes $ input $ output),
  Term.info "udhcp" ~doc ~man ~version:"%%VERSION%%"

let () = match Term.eval term with
  | `Error _ -> exit 1
  | _        -> ()
