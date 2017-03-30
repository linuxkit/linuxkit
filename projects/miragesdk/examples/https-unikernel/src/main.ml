open Lwt.Infix
open Capnp_rpc_lwt

let switch = Lwt_switch.create ()

let socket_pair ~switch =
  let client, server = Unix.(socketpair PF_UNIX SOCK_STREAM 0) in
  Lwt_switch.add_hook (Some switch) (fun () ->
      Unix.close client;
      Unix.close server;
      Lwt.return ()
    );
  (Endpoint.of_socket ~switch client, Endpoint.of_socket ~switch server)

let store_to_http, http_to_store = socket_pair ~switch
let http_to_tls, tls_to_http = socket_pair ~switch

let () =
  Common.init_logging ();
  Lwt_main.run begin
    begin
      Store.service () >>= fun service ->
      let tags = Logs.Tag.add Common.Actor.tag (`Green, "Store ") Logs.Tag.empty in
      let _ : CapTP.t = CapTP.of_endpoint ~offer:service ~tags ~switch store_to_http in
      Lwt.return ()
    end
    >>= fun () ->
    begin
      let tags = Logs.Tag.add Common.Actor.tag (`Red, "HTTP  ") Logs.Tag.empty in
      let store = CapTP.bootstrap (CapTP.of_endpoint ~tags ~switch http_to_store) in
      let service = Http_server.service store in
      let _ : CapTP.t = CapTP.of_endpoint ~offer:service ~tags ~switch http_to_tls in
      Lwt.return ()
    end
    >>= fun () ->
    Tls_terminator.init ~switch ~to_http:tls_to_http
  end
