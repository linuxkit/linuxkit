open Lwt.Infix
open Sdk

let src = Logs.Src.create "dhcp-client" ~doc:"DHCP client"
module Log = (val Logs.src_log src : Logs.LOG)

let failf fmt = Fmt.kstrf Lwt.fail_with fmt


module Handlers = struct

  (* System handlers *)

  let contents_of_diff = function
    | `Added (_, `Contents (v, _))
    | `Updated (_, (_, `Contents (v, _))) -> Some v
    | _ -> None

  let ip t =
    Ctl.KV.watch_key t ["ip"] (fun diff ->
        match contents_of_diff diff with
        | Some ip ->
          Log.info (fun l -> l "SET IP to %s" ip);
          Lwt.return ()
        | _ ->
          Lwt.return ()
      )

  let handlers = [
    ip;
  ]

  let watch path =
    Ctl.v path >>= fun db ->
    Lwt_list.map_p (fun f -> f db) handlers >>= fun _ ->
    let t, _ = Lwt.task () in
    t

end

external bpf_filter: unit -> string = "bpf_filter"

let run () cmd ethif path =
  Lwt_main.run (
    let net = Init.rawlink ~filter:(bpf_filter ()) ethif in
    let routes = [
      "/ip";
      "/domain";
      "/search";
      "/mtu";
      "/nameservers/*"
    ] in
    Ctl.v "/data" >>= fun ctl ->
    let fd = Init.(Fd.fd @@ Pipe.(priv ctl)) in
    let ctl () = Ctl.Server.listen ~routes ctl fd in
    let handlers () = Handlers.watch path in
    Init.run ~net ~ctl ~handlers cmd
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

let ctl = string_of_int Init.(Fd.to_int Pipe.(calf ctl))
let net = string_of_int Init.(Fd.to_int Pipe.(calf net))

let cmd =
  (* FIXME: use runc isolation
   let default_cmd = [
    "/usr/bin/runc"; "--"; "run";
    "--bundle"; "/containers/images/000-dhcp-client";
    "dhcp-client"
  ] in
  *)
  let default_cmd = [
    "/dhcp-client-calf"; "--ctl="^ctl; "--net="^net
  ] in
  let doc =
    Arg.info ~docv:"CMD" ~doc:"Command to run the calf process." ["cmd"]
  in
  Arg.(value & opt (list ~sep:' ' string) default_cmd & doc)

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
