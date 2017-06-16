let () = Common.init_logging ()

let main store_socket http_socket =
  Lwt_main.run begin
    Lwt_switch.with_switch @@ fun switch ->
    let store = Common.connect ~switch store_socket in
    let http = Http_server.service store in
    Common.listen ~switch ~offer:http http_socket
  end

open Cmdliner

let store =
  let doc = "The database store to use" in
  Arg.(required @@ opt (some string) None @@ info ~doc ~docv:"STORE" ["store"])

let http =
  let doc = "The http socket to provide" in
  Arg.(required @@ pos 0 (some string) None @@ info ~doc ~docv:"HTTP" [])

let cmd =
  Term.(const main $ store $ http), Term.info "http"

let () = Term.(exit @@ eval cmd)
