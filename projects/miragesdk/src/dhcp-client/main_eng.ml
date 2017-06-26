open Lwt.Infix

module Flow = Sdk.Flow.Fd
module Time = Sdk.Time.Local
module Net = Sdk.Net.Client(Flow)
module Act = Sdk.Host.Client(Flow)

module Main = Engine.Make(Time)(Net)(Act)

let start ~net ~act =
  Lwt_switch.with_switch @@ fun switch ->
  Flow.connect net >>= fun net ->
  Net.connect ~switch net >>= fun net ->
  Flow.connect act >>= fun act ->
  Act.connect ~switch act >>= fun act ->
  Main.start () net act

let run () net act = Lwt_main.run (start ~net ~act)

open Cmdliner

let net =
  let doc =
    Arg.info
      ~docv:"FD"
      ~doc:"The file descriptor to use to connect to the network proxy."
      ["e"; "engine"]
  in
  Arg.(value & opt int 3 & doc)

let act =
  let doc =
    Arg.info
      ~docv:"FD"
      ~doc:"The file descriptor to use to connect to the host actuator."
      ["a"; "actuator"]
  in
  Arg.(value & opt int 4 & doc)

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

let run =
  Term.(const run $ setup_log $ net $ act),
  Term.info "dhcp-client-engine" ~version:"%%VERSION%%"

let () = match Term.eval run with
  | `Error _ -> exit 1
  | `Ok () |`Help |`Version -> exit 0
