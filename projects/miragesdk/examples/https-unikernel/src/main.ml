(** Run all the services together in a single process, communicating over Unix-domain sockets. *)

open Lwt.Infix
open Capnp_rpc_lwt

let switch = Lwt_switch.create ()

let socket_pair ~switch =
  let client, server = Lwt_unix.(socketpair PF_UNIX SOCK_STREAM 0) in
  (Capnp_rpc_unix.endpoint_of_socket ~switch client,
   Capnp_rpc_unix.endpoint_of_socket ~switch server)

let store_to_http, http_to_store = socket_pair ~switch
let http_to_tls, tls_to_http = socket_pair ~switch

let () =
  Logging.init ();
  Lwt_main.run begin
    begin
      Store.local () >>= fun service ->
      let tags = Logs.Tag.add Logging.Actor.tag (`Green, "Store ") Logs.Tag.empty in
      let _ : CapTP.t = CapTP.connect ~offer:service ~tags ~switch store_to_http in
      Lwt.return ()
    end
    >>= fun () ->
    begin
      let tags = Logs.Tag.add Logging.Actor.tag (`Red, "HTTP  ") Logs.Tag.empty in
      let store = CapTP.bootstrap (CapTP.connect ~tags ~switch http_to_store) in
      let service = Http_server.local store in
      let _ : CapTP.t = CapTP.connect ~offer:service ~tags ~switch http_to_tls in
      Lwt.return ()
    end
    >>= fun () ->
    Tls_terminator.init ~switch ~to_http:tls_to_http
  end
