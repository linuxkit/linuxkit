open Lwt.Infix

let src = Logs.Src.create "init" ~doc:"Init steps"
module Log = (val Logs.src_log src : Logs.LOG)

(* FIXME: to avoid linking with gmp *)
module IO = struct
  type ic = unit
  type oc = unit
  type ctx = unit
  let with_connection ?ctx:_ _uri ?init:_ _f = Lwt.fail_with "not allowed"
  let read_all _ic = Lwt.fail_with "not allowed"
  let read_exactly _ic _n = Lwt.fail_with "not allowed"
  let write _oc _buf = Lwt.fail_with "not allowed"
  let flush _oc = Lwt.fail_with "not allowed"
  let ctx () = Lwt.return_none
end

(* FIXME: we don't use Irmin_unix.Git.FS.KV to avoid linking with gmp *)
module Store = Irmin_git.FS.KV(IO)(Inflator)(Io_fs)
module KV = Store(Irmin.Contents.String)

let v path =
  let config = Irmin_git.config path in
  KV.Repo.v config >>= fun repo ->
  KV.of_branch repo "calf"

let set_listen_dir_hook () =
  Irmin.Private.Watch.set_listen_dir_hook Irmin_watcher.hook

module HTTP = struct

  module Wm = struct
    module Rd = Webmachine.Rd
    include Webmachine.Make(Cohttp_lwt_unix.Server.IO)
  end

  let with_key rd f =
    match KV.Key.of_string rd.Wm.Rd.dispatch_path with
    | Ok x    -> f x
    | Error _ -> Wm.respond 404 rd

  let infof fmt =
    Fmt.kstrf (fun msg () ->
        let date = Int64.of_float (Unix.gettimeofday ()) in
        Irmin.Info.v ~date ~author:"calf" msg
      ) fmt

  let ok = "{\"status\": \"ok\"}"

  class item db = object(self)

    inherit [Cohttp_lwt_body.t] Wm.resource

    method private of_string rd =
      Cohttp_lwt_body.to_string rd.Wm.Rd.req_body >>= fun value ->
      with_key rd (fun key ->
          let info = infof "Updating %a" KV.Key.pp key in
          KV.set db ~info key value >>= fun () ->
          let resp_body = `String ok in
          let rd = { rd with Wm.Rd.resp_body } in
          Wm.continue true rd
        )

    method private to_string rd =
      with_key rd (fun key ->
          KV.find db key >>= function
          | Some value -> Wm.continue (`String value) rd
          | None       -> assert false
        )

    method resource_exists rd =
      with_key rd (fun key ->
          KV.mem db key >>= fun mem ->
          Wm.continue mem rd
        )

    method allowed_methods rd =
      Wm.continue [`GET; `HEAD; `PUT; `DELETE] rd

    method content_types_provided rd =
      Wm.continue [
        "plain", self#to_string
      ] rd

    method content_types_accepted rd =
      Wm.continue [
        "plain", self#of_string
      ] rd

    method delete_resource rd =
      with_key rd (fun key ->
          let info = infof "Deleting %a" KV.Key.pp key in
          KV.remove db ~info key >>= fun () ->
          let resp_body = `String ok in
          Wm.continue true { rd with Wm.Rd.resp_body }
        )
  end

  let v db routes =
    let routes = List.map (fun r -> r, fun () -> new item db) routes in
    let callback (_ch, _conn) request body =
      let open Cohttp in
      (Wm.dispatch' routes ~body ~request >|= function
        | None        -> (`Not_found, Header.init (), `String "Not found", [])
        | Some result -> result)
      >>= fun (status, headers, body, path) ->
      Log.info (fun l ->
          l "%d - %s %s"
            (Code.code_of_status status)
            (Code.string_of_method (Request.meth request))
            (Uri.path (Request.uri request)));
      Log.debug (fun l -> l "path=%a" Fmt.(Dump.list string) path);
      (* Finally, send the response to the client *)
      Cohttp_lwt_unix.Server.respond ~flush:true ~headers ~body ~status ()
    in
    (* create the server and handle requests with the function defined above *)
    let conn_closed (_, conn) =
      Log.info (fun l ->
          l "connection %s closed\n%!" (Cohttp.Connection.to_string conn))
    in
    Cohttp_lwt_unix.Server.make ~callback ~conn_closed ()
end

let serve ~routes db fd =
  let http = HTTP.v db routes in
  let on_exn e = Log.err (fun l -> l "ERROR: %a" Fmt.exn e) in
  Cohttp_lwt_unix.Server.create ~on_exn ~mode:(`Fd fd) http
