(** The TLS terminator implementation.
    Listens for TLS connections on a port and forwards the plaintext flow to the HTTP service. *)

open Lwt.Infix
open Capnp_rpc_lwt

let run ~port ~http_service =
  let tls_config : Conduit_lwt_unix.server_tls_config =
    `Crt_file_path "tls-secrets/server.crt",
    `Key_file_path "tls-secrets/server.key",
    `No_password,
    `Port port
  in
  let mode = `TLS tls_config in
  Logs.info (fun f -> f "Listening on https port %d" port);
  Conduit_lwt_unix.(serve ~ctx:default_ctx) ~mode (fun _flow ic oc ->
      Logs.info (fun f -> f "Got new TLS connection");
      let flow_obj = Rpc.Flow.local ic oc in
      Rpc.Http.accept http_service flow_obj >|= fun () ->
      Capability.dec_ref flow_obj
    )

let init ~switch ~to_http =
  let tags = Logs.Tag.add Logging.Actor.tag (`Blue, "TLS   ") Logs.Tag.empty in
  let http_service = CapTP.bootstrap (CapTP.connect ~tags ~switch to_http) in
  run ~http_service ~port:8443
