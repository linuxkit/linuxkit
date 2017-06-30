open Astring
open Cmdliner

let socket =
  let doc =
    Arg.info ~docv:"PATH"
      ~doc:"Socket to communicate with the FDD server." ["s"; "socket"]
  in
  Arg.(value & opt string "/var/run/fdd.sock" doc)

let share =
  let doc =
    Arg.info ~docv:"PATH" ~doc:"The path to use to share the socketpair." []
  in
  Arg.(required & pos 0 (some string) None doc)

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

let run f = Lwt_main.run f

let init =
  let f socket () = run (Init.f socket) in
  Term.(const f $ socket $ setup_log),
  Term.info "init" ~doc:"Start the FDD server"

let test =
  let f share () = run (Test.f share) in
  Term.(const f $ share $ setup_log),
  Term.info "test" ~doc:"Test a socketpair share."

let share =
  let f socket share () = run (Share.f ~socket ~share) in
  Term.(const f $ socket $ share $ setup_log),
  Term.info "share" ~doc:"Share a new socketpair on a given unix domain socket."

let exec =
  let dup =
    let parse str = match String.cuts ~sep:":" str with
      | [] | [_] ->
        Error (`Msg ("A valid share map should have the form \
                      <path>:<fd-number>[:fd-number]*"))
      | s :: fds  -> Ok (s, List.map int_of_string fds)
    in
    let pp ppf (name, fds) =
      Fmt.pf ppf "%s:%a" name Fmt.(list ~sep:(unit ":") int) fds
    in
    Arg.conv (parse, pp)
  in
  let dups =
    let doc =
      Arg.info ~docv:"MAP" ~doc:
        "Maps of socketpairs/local fds in the form \
         <path>:<fd-number>[:fd-number]*,..."
        ["m";"map"]
    in
    Arg.(value & opt (list dup) [] doc)
  in
  let cmd =
    let doc = Arg.info ~docv:"COMMAND" ~doc:"The command to execute" [] in
    Arg.(non_empty & pos_all string [] doc)
  in
  let f dups cmd () = run (Exec.f dups cmd) in
  Term.(const f $ dups $ cmd $ setup_log),
  Term.info "exec"
    ~doc:"Execute a command with a side of the socketpair pre-opened on the \
          specified files descriptors."

let default =
  let usage () =
    Fmt.pr "usage: fdd [--version]\n\
           \           [--help]\n\
           \           <command> [<args>]\n\
            \n\
            The most commonly used subcommands are:\n\
           \    init        start a new FDD server\n\
           \    share       share a new socketpair\n\
           \    test        test a socketpair share\n\
            \n\
            See `fdd help <command>` for more information on a specific \
            command.\n%!"
  in
  Term.(const usage $ const ()),
  Term.info "fdd" ~version:"%%VERSION%%"
    ~doc:"Share socketpairs over unix domain sockets."

let cmds = [
  init;
  share;
  test;
  exec;
]

let () = match Term.eval_choice default cmds with
  | `Error _ -> exit 1
  | `Ok () |`Help |`Version -> exit 0
