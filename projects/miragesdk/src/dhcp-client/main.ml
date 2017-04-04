open Lwt.Infix
open Sdk
open Astring

let src = Logs.Src.create "dhcp-client" ~doc:"DHCP client"
module Log = (val Logs.src_log src : Logs.LOG)

let failf fmt = Fmt.kstrf Lwt.fail_with fmt

let run fmt =
  Fmt.kstrf (fun str ->
      match Sys.command str with
      | 0 -> Lwt.return ()
      | i -> Fmt.kstrf Lwt.fail_with "%S exited with code %d" str i
    ) fmt

module Handlers = struct

  (* System handlers *)

  let contents_of_diff = function
    | `Added (_, `Contents (v, _))
    | `Updated (_, (_, `Contents (v, _))) -> Some v
    | _ -> None

  let ip ~ethif t =
    Ctl.KV.watch_key t ["ip"] (fun diff ->
        match contents_of_diff diff with
        | None    -> Lwt.return_unit
        | Some ip ->
          let ip = String.trim ip in
          Log.info (fun l -> l "SET IP to %s" ip);
          (* FIXME: use language bindings to netlink instead *)
          run "ifconfig %s %s netmask 255.255.255.0" ethif ip
          (* run "ip addr add %s/24 dev %s" ip ethif *)
      )

  let gateway t =
    Ctl.KV.watch_key t ["gateway"] (fun diff ->
        match contents_of_diff diff with
        | None    -> Lwt.return_unit
        | Some gw ->
          let gw = String.trim gw in
          Log.info (fun l -> l "SET GATEWAY to %s" gw);
          (* FIXME: use language bindings to netlink instead *)
          run "ip route add default via %s" gw
      )

  let handlers ~ethif = [
    ip ~ethif;
    gateway;
  ]

  let watch ~ethif path =
    Ctl.v path >>= fun db ->
    Lwt_list.map_p (fun f -> f db) (handlers ethif) >>= fun _ ->
    let t, _ = Lwt.task () in
    t

end

external bpf_filter: unit -> string = "bpf_filter"

let t = Init.Pipe.v ()

let ctl = string_of_int Init.(Fd.to_int Pipe.(calf @@ ctl t))
let net = string_of_int Init.(Fd.to_int Pipe.(calf @@ net t))
let default_cmd = [
  "/dhcp-client-calf"; "--ctl="^ctl; "--net="^net
]

(* FIXME: use runc isolation
   let default_cmd = [
    "/usr/bin/runc"; "--"; "run";
    "--bundle"; "/containers/images/000-dhcp-client";
    "dhcp-client"
  ] in
  *)

let read_cmd file =
  if Sys.file_exists file then
    let ic = open_in_bin file in
    let line = input_line ic in
    String.cuts ~sep:" " line
  else
    failwith ("Cannot read " ^ file)

 let run () cmd ethif path =
  let cmd = match cmd with
    | None   -> default_cmd
    | Some f -> read_cmd f
  in
  Lwt_main.run (
    let net = Init.rawlink ~filter:(bpf_filter ()) ethif in
    let routes = [
      "/ip";
      "/gateway";
      "/domain";
      "/search";
      "/mtu";
      "/nameservers/*"
    ] in
    Ctl.v "/data" >>= fun ctl ->
    let fd = Init.(Fd.fd @@ Pipe.(priv @@ ctl t)) in
    let ctl () = Ctl.Server.listen ~routes ctl fd in
    let handlers () = Handlers.watch ~ethif path in
    Init.run t ~net ~ctl ~handlers cmd
  )

(* CLI *)

open Cmdliner

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

let cmd =
  let doc =
    Arg.info ~docv:"CMD" ~doc:"Command to run the calf process." ["cmd"]
  in
  Arg.(value & opt (some string) None & doc)

let ethif =
  let doc =
    Arg.info ~docv:"NAME" ~doc:"The interface to listen too." ["ethif"]
  in
  Arg.(value & opt string "eth0" & doc)

let path =
  let doc =
    Arg.info ~docv:"DIR"
      ~doc:"The directory where control state will be stored." ["path"]
  in
  Arg.(value & opt string "/data" & doc)

let run =
  Term.(const run $ setup_log $ cmd $ ethif $ path),
  Term.info "dhcp-client" ~version:"0.0"

let () = match Term.eval run with
  | `Error _ -> exit 1
  | `Ok () |`Help |`Version -> exit 0
