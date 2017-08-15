(** Accepts connections (over Cap'n Proto) and implements the HTTP protocol. *)

let src = Logs.Src.create "web.http" ~doc:"HTTP engine for web server"
module Log = (val Logs.src_log src: Logs.LOG)

open Capnp_rpc_lwt
open Lwt.Infix
open Astring

module IO = struct
  type 'a t = 'a Lwt.t
  let (>>=) = Lwt.bind
  let return = Lwt.return

  type ic = Lwt_io.input_channel
  type oc = Lwt_io.output_channel
  type conn = Rpc.Flow.flow

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

(* Handle one HTTP request *)
let handle_request store _conn req _body =
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
    begin Rpc.Store.get store path >>= function
    | Some body -> Server.respond_string ~status:`OK ~body ()
    | None -> Server.respond_not_found ~uri ()
    end
  | m ->
    let body = Fmt.strf "Bad method %S" (Code.string_of_method m) in
    Server.respond_error ~status:`Bad_request ~body ()

module Remote_flow_unix = Mirage_flow_unix.Make(Rpc.Flow)

let local store =
  let handle_http_connection = Server.callback (Server.make ~callback:(handle_request store) ()) in
  Rpc.Http.local (fun flow ->
      Log.info (fun f -> f "Handing new connection");
      handle_http_connection flow (Remote_flow_unix.ic flow) (Remote_flow_unix.oc flow) >>= fun () ->
      Capability.dec_ref flow;
      Lwt.return_unit
    )
