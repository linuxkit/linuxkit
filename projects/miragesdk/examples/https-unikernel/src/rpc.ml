open Lwt.Infix
open Capnp_rpc_lwt

module Api = Proto.MakeRPC(Capnp_rpc_lwt)

module Flow = struct
  let local ic oc =
    let module F = Api.Service.Flow in
    F.local @@ object
      inherit F.service

      method read_impl _ release_param_caps =
        release_param_caps ();
        Service.return_lwt (fun () ->
            Lwt_io.read ~count:4096 ic >|= fun data ->
            let open F.Read in
            let resp, results = Service.Response.create Results.init_pointer in
            Results.data_set results data;
            Ok resp
          )

      method write_impl req release_param_caps =
        release_param_caps ();
        let open F.Write in
        let data = Params.data_get req in
        Service.return_lwt (fun () ->
            Lwt_io.write oc data >>= fun () ->
            Lwt.return (Ok (Service.Response.create_empty ()))
          )
    end

  module Flow = Api.Client.Flow

  type buffer = Cstruct.t
  type flow = Flow.t Capability.t
  type error = [`Capnp of Capnp_rpc.Error.t]
  type write_error = [Mirage_flow.write_error | `Capnp of Capnp_rpc.Error.t]
  type 'a io = 'a Lwt.t

  let read t =
    let open Flow.Read in
    let req = Capability.Request.create_no_args () in
    Capability.call_for_value_exn t method_id req >>= fun resp ->
    match Results.data_get resp with
    | "" -> Lwt.return (Ok `Eof)
    | data -> Lwt.return (Ok (`Data (Cstruct.of_string data)))

  let write t data =
    let open Flow.Write in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.data_set p (Cstruct.to_string data);
    Capability.call_for_unit_exn t method_id req >>= fun () ->
    Lwt.return (Ok ())

  let writev t data =
    write t (Cstruct.concat data)

  let close _ = failwith "TODO: close"

  let pp_error f = function
    | `Capnp e -> Capnp_rpc.Error.pp f e
    | `Closed -> Fmt.string f "Closed"

  let pp_write_error f = function
    | `Capnp e -> Capnp_rpc.Error.pp f e
    | #Mirage_flow.write_error as e -> Mirage_flow.pp_write_error f e

end

module Store = struct
  (* The Cap'n'Proto service interface we expose. *)
  let local lookup =
    let module Store = Api.Service.Store in
    Store.local @@ object
      inherit Store.service

      method get_impl req release_param_caps =
        let open Store.Get in
        let path = Params.path_get_list req in
        release_param_caps ();
        Service.return_lwt (fun () ->
            let resp, results = Service.Response.create Results.init_pointer in
            begin
              lookup path >|= function
              | Some data -> Results.ok_set results data
              | None -> Results.not_found_set results
            end
            >>= fun () ->
            Lwt.return (Ok resp)
          )
    end

  module Store = Api.Client.Store

  type t = Store.t Capability.t

  (* Make a Cap'n'Proto call to the store service *)
  let get t path =
    let open Store.Get in
    let req, p = Capability.Request.create Params.init_pointer in
    ignore (Params.path_set_list p path);
    Capability.call_for_value_exn t method_id req >>= fun resp ->
    let open Api.Reader.Store in
    match GetResults.get resp with
    | GetResults.NotFound -> Lwt.return None
    | GetResults.Ok data -> Lwt.return (Some data)
    | GetResults.Undefined _ -> failwith "Protocol error: bad msg type"
end

module Http = struct
  let local handle_connection =
    let module HttpServer = Api.Service.HttpServer in
    HttpServer.local @@ object
      inherit HttpServer.service

      method accept_impl req release_param_caps =
        let open HttpServer.Accept in
        let flow = Params.connection_get req in
        release_param_caps ();
        match flow with
        | None -> Service.fail "No connection!"
        | Some c ->
          Service.return_lwt (fun () ->
              handle_connection c >|= fun () ->
              Ok (Service.Response.create_empty ())
            )
    end

  module HttpServer = Api.Client.HttpServer

  type t = HttpServer.t Capability.t

  let accept t flow =
    let open HttpServer.Accept in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.connection_set p (Some flow);
    Capability.call_for_unit t method_id req >|= function
    | Ok () -> ()
    | Error e -> Logs.warn (fun f -> f "Error from HTTP server: %a" Capnp_rpc.Error.pp e)
end
