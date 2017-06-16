open Lwt.Infix
open Capnp_rpc_lwt

let make_flow _flow ic oc =
  Api.Builder.Flow.local @@
    object (_ : Api.Builder.Flow.service)
      method read _ =
        Service.return_lwt (fun () ->
          Lwt_io.read ~count:4096 ic >|= fun data ->
          let module R = Api.Builder.Flow.Read_results in
          let resp, results = Service.Response.create R.init_pointer in
          R.data_set results data;
          Ok resp
        )

      method write req =
        let module R = Api.Reader.Flow.Write_params in
        let p = R.of_payload req in
        let data = R.data_get p in
        Service.return_lwt (fun () ->
          Lwt_io.write oc data >>= fun () ->
          Lwt.return (Ok (Service.Response.create_empty ()))
        )
    end

let handle ~http_service flow =
  let proxy = new Api.Reader.HttpServer.client http_service in
  let module P = Api.Builder.HttpServer.Accept_params in
  let req, p = Capability.Request.create P.init_pointer in
  P.connection_set p (Some (Capability.Request.export req flow));
  Capability.call_for_value proxy#accept req >|= function
  | Ok _ -> ()
  | Error e -> Logs.warn (fun f -> f "Error from HTTP server: %a" Capnp_rpc.Error.pp e)

let run ~port ~http_service =
  let tls_config : Conduit_lwt_unix.server_tls_config =
    `Crt_file_path "tls-secrets/server.crt",
    `Key_file_path "tls-secrets/server.key",
    `No_password,
    `Port port
  in
  let mode = `TLS tls_config in
  Logs.info (fun f -> f "Listening on https port %d" port);
  Conduit_lwt_unix.(serve ~ctx:default_ctx) ~mode (fun flow ic oc ->
      Logs.info (fun f -> f "Got new TLS connection");
      let flow_obj = make_flow flow ic oc in
      handle ~http_service flow_obj
    )

let init ~switch ~to_http =
  let tags = Logs.Tag.add Common.Actor.tag (`Blue, "TLS   ") Logs.Tag.empty in
  let http_service = CapTP.bootstrap (CapTP.of_endpoint ~tags ~switch to_http) in
  run ~http_service ~port:8443
