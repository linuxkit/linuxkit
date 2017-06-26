open Lwt.Infix

module Flow = Sdk.Flow.Fd
module Act = Sdk.Host.Client(Flow)
module Net = Network.Make(Act)

module Main = Sdk.Net.Server(Flow)(Net)

let start ~eng ~act =
  Lwt_switch.with_switch @@ fun switch ->
  Flow.connect act >>= fun act ->
  Act.connect ~switch act >>= fun act ->
  Flow.connect eng >>= fun eng ->
  Net.connect act >>= fun net ->
  Main.listen ~switch (Main.service net) eng;
  fst (Lwt.task ())

let run () eng act = Lwt_main.run (start ~eng ~act)

open Cmdliner

let eng =
  let doc =
    Arg.info
      ~docv:"FD"
      ~doc:"The file descriptor to use to connect to the DHCP client engine."
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
  Term.(const run $ setup_log $ eng $ act),
  Term.info "dhcp-client-network" ~version:"%%VERSION%%"

let () = match Term.eval run with
  | `Error _ -> exit 1
  | `Ok () |`Help |`Version -> exit 0
