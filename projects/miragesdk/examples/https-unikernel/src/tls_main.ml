let () = Common.init_logging ()

let main http_socket port =
  Lwt_main.run begin
    Lwt_switch.with_switch @@ fun switch ->
    let http_service = Common.connect ~switch http_socket in
    Tls_terminator.run ~http_service ~port
  end

open Cmdliner

let http =
  let doc = "The HTTP service to use" in
  Arg.(required @@ opt (some string) None @@ info ~doc ~docv:"HTTP" ["http"])

let tls =
  let doc = "The TLS port on which to listen for incoming connections" in
  Arg.(value @@ opt int 8443 @@ info ~doc ~docv:"PORT" ["port"])

let cmd =
  Term.(const main $ http $ tls), Term.info "tls"

let () = Term.(exit @@ eval cmd)
