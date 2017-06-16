let src = Logs.Src.create "web.http" ~doc:"HTTP engine for web server"
module Log = (val Logs.src_log src: Logs.LOG)

open Capnp_rpc_lwt
open Lwt.Infix
open Astring

module Remote_flow = struct
  type buffer = Cstruct.t
  type flow = Api.Reader.Flow.t Capability.t
  type error = [`Capnp of Capnp_rpc.Error.t]
  type write_error = [Mirage_flow.write_error | `Capnp of Capnp_rpc.Error.t]
  type 'a io = 'a Lwt.t

  let create x = x

  let read t =
    let module R = Api.Reader.Flow.Read_results in
    let req = Capability.Request.create_no_args () in
    let proxy = new Api.Reader.Flow.client t in
    Capability.call_for_value_exn proxy#read req >>= fun resp ->
    let p = R.of_payload resp in
    let data = R.data_get p in
    Lwt.return (Ok (`Data (Cstruct.of_string data)))

  let write t data =
    let module P = Api.Builder.Flow.Write_params in
    let req, p = Capability.Request.create P.init_pointer in
    let proxy = new Api.Reader.Flow.client t in
    P.data_set p (Cstruct.to_string data);
    Capability.call_for_value_exn proxy#write req >>= fun _ ->
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

module IO = struct
  type 'a t = 'a Lwt.t
  let (>>=) = Lwt.bind
  let return = Lwt.return

  type ic = Lwt_io.input_channel
  type oc = Lwt_io.output_channel
  type conn = Remote_flow.flow

  let read_line ic =
    Lwt_io.read_line_opt ic

  let read ic count =
    let count = min count Sys.max_string_length in
    Lwt_io.read ~count ic

  let write oc buf =
    Lwt_io.write oc buf

  let flush oc =
    Lwt_io.flush oc
end

module Server = Cohttp_lwt.Make_server(IO)

type t = Api.Reader.Store.t Capability.t

(* Make a Cap'n'Proto call to the store service *)
let get t path =
  let module P = Api.Builder.Store.Get_params in
  let req, p = Capability.Request.create P.init_pointer in
  ignore (P.path_set_list p path);
  let proxy = new Api.Reader.Store.client t in
  Capability.call_for_value_exn proxy#get req >>= fun resp ->
  let open Api.Reader.Store in
  match GetResults.get (GetResults.of_payload resp) with
  | GetResults.NotFound -> Lwt.return None
  | GetResults.Ok data -> Lwt.return (Some data)
  | GetResults.Undefined _ -> failwith "Protocol error: bad msg type"

(* Handle HTTP requests *)
let callback t _conn req _body =
  let open Cohttp in
  let uri = Request.uri req in
  Log.info (fun f -> f "HTTP request for %a" Uri.pp_hum uri);
  match Request.meth req with
  | `GET ->
    let path = String.cuts ~empty:false ~sep:"/" (Uri.path uri) in
    let path =
      match path with
      | [] -> ["index.html"]
      | p -> p
    in
    begin get t path >>= function
    | Some body -> Server.respond_string ~status:`OK ~body ()
    | None -> Server.respond_not_found ~uri ()
    end
  | m ->
    let body = Fmt.strf "Bad method %S" (Code.string_of_method m) in
    Server.respond_error ~status:`Bad_request ~body ()

let callback t = Server.callback (Server.make ~callback:(callback t) ())

module Remote_flow_unix = Mirage_flow_unix.Make(Remote_flow)

let handle_connection store c =
  Log.info (fun f -> f "Handing new connection");
  let flow = Remote_flow.create c in
  callback store flow (Remote_flow_unix.ic flow) (Remote_flow_unix.oc flow) >>= fun () ->
  Capability.dec_ref c;
  Lwt.return_unit

let service store =
  Api.Builder.HttpServer.local @@
    object (_ : Api.Builder.HttpServer.service)
      method accept req =
        Log.info (fun f -> f "Handing new connection");
        let module P = Api.Reader.HttpServer.Accept_params in
        let p = P.of_payload req in
        match P.connection_get p with
        | None -> Service.fail "No connection!"
        | Some i ->
          let c = Payload.import req i in
          Service.return_lwt (fun () ->
            handle_connection store c >|= fun () ->
            Ok (Service.Response.create_empty ())
          )
    end
