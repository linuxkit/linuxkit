open Lwt.Infix

let src = Logs.Src.create "dhcp-client/actuator"
module Log = (val Logs.src_log src : Logs.LOG)

module Flow = Sdk.Flow.Fd
module Host = Sdk.Host.Local
module N = Sdk.Host.Server(Flow)(Host)
module E = Sdk.Host.Server(Flow)(Host)

let start ~intf ~net ~eng =
  Lwt_switch.with_switch @@ fun switch ->
  Flow.connect net >>= fun net ->
  Flow.connect eng >>= fun eng ->
  Host.connect intf >>= fun host ->
  N.listen ~switch (N.service host) net;
  E.listen ~switch (E.service host) eng;
  fst (Lwt.task ())

let run () intf net eng = Lwt_main.run (start ~intf ~net ~eng)

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

let intf =
  let doc =
    Arg.info ~docv:"INTF" ~doc:"The interface to listen too."
      ["e"; "ethif"]
  in
  Arg.(value & opt string "eth0" doc)

let eng =
  let doc =
    Arg.info
      ~docv:"FD"
      ~doc:"The file descriptor to use to connect to the DHCP client engine."
      ["e"; "engine"]
  in
  Arg.(value & opt int 3 & doc)

let net =
  let doc =
    Arg.info
      ~docv:"FD"
      ~doc:"The file descriptor to use to connect to the network proxy."
      ["n"; "network"]
  in
  Arg.(value & opt int 4 & doc)

let run =
  Term.(const run $ setup_log $ intf $ net $ eng),
  Term.info "dhcp-client-actuator" ~version:"%%VERSION%%"

let () = match Term.eval run with
  | `Error _ -> exit 1
  | `Ok () |`Help |`Version -> exit 0
