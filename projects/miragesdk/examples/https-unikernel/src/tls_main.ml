(** Run the TLS terminator as a stand-alone Unix process. *)

let () = Logging.init ()

let main http_addr port =
  Lwt_main.run begin
    let http_service = Capnp_rpc_unix.connect http_addr in
    Tls_terminator.run ~http_service ~port
  end

open Cmdliner

let http =
  let doc = "The HTTP service to use" in
  Arg.(required @@ opt (some Capnp_rpc_unix.Connect_address.conv) None @@ info ~doc ~docv:"HTTP" ["http"])

let tls =
  let doc = "The TLS port on which to listen for incoming connections" in
  Arg.(value @@ opt int 8443 @@ info ~doc ~docv:"PORT" ["port"])

let cmd =
  Term.(const main $ http $ tls), Term.info "tls"

let () = Term.(exit @@ eval cmd)
