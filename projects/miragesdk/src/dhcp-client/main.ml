open Lwt.Infix

module Act = Sdk.Host.Local
module Net = Network.Make(Act)
module Eng = Engine.Make(Sdk.Time.Local)(Net)(Act)

let main intf =
  Act.connect intf >>= fun act ->
  Net.connect act >>= fun net ->
  Eng.start () net act

let run () intf = Lwt_main.run (main intf)

open Cmdliner

let intf =
  let doc =
    Arg.info ~docv:"INTF" ~doc:"The interface to listen too."
      ["e"; "ethif"]
  in
  Arg.(value & opt string "eth0" doc)

(* FIXME: use SDK to write logs *)
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
  Term.(const run $ setup_log $ intf),
  Term.info "dhcp-client" ~version:"%%VERSION%%"

let () = match Term.eval run with
  | `Error _ -> exit 1
  | `Ok () |`Help |`Version -> exit 0
