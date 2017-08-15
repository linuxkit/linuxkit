(** Run the HTTP service as a stand-alone Unix process. *)

let () = Logging.init ()

let main store_addr http_addr =
  Lwt_main.run begin
    let store = Capnp_rpc_unix.connect store_addr in
    let http = Http_server.local store in
    Capnp_rpc_unix.serve ~offer:http http_addr
  end

open Cmdliner

let store =
  let doc = "The database store to use" in
  Arg.(required @@ opt (some Capnp_rpc_unix.Connect_address.conv) None @@ info ~doc ~docv:"STORE" ["store"])

let http =
  let doc = "The http socket to provide" in
  Arg.(required @@ pos 0 (some Capnp_rpc_unix.Listen_address.conv) None @@ info ~doc ~docv:"HTTP" [])

let cmd =
  Term.(const main $ store $ http), Term.info "http"

let () = Term.(exit @@ eval cmd)
