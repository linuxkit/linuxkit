open Lwt.Infix
open Capnp_rpc_lwt

let src = Logs.Src.create "init" ~doc:"Init steps"
module Log = (val Logs.src_log src : Logs.LOG)

(* FIXME: to avoid linking with gmp *)
module No_IO = struct
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
module Store = Irmin_git.FS.KV(No_IO)(Inflator)(Io_fs)
module KV = Store(Irmin.Contents.String)

let pp_path = Fmt.(brackets (list ~sep:(const string "/") string))

let v path =
  let config = Irmin_git.config path in
  KV.Repo.v config >>= fun repo ->
  KV.of_branch repo "calf"

let () =
  Irmin.Private.Watch.set_listen_dir_hook
    (fun _ _ _ -> Lwt.return (fun () -> Lwt.return_unit))
    (* FIXME: inotify need some unknown massaging. *)
    (* Irmin_watcher.hook *)

module C = Mirage_channel_lwt.Make(Mirage_flow_lwt)

exception Undefined_field of int

let errorf fmt =
  Fmt.kstrf (fun x -> Error (`Msg x)) fmt

module Client = struct

  module C = Api.Reader.Ctl

  type error = [`Msg of string]
  let pp_error ppf (`Msg s) = Fmt.string ppf s

  type t = C.t Capability.t

  let read t path =
    let module P = Api.Builder.Ctl.Read_params in
    let module R = Api.Reader.Response in
    let req, p = Capability.Request.create P.init_pointer in
    P.path_set_list p path |> ignore;
    Capability.call_for_value t C.read_method req >|= function
    | Error e -> errorf "error: read(%a) -> %a" pp_path path Capnp_rpc.Error.pp e
    | Ok r ->
      let r = R.of_payload r in
      match R.get r with
      | R.Ok data -> Ok (Some data)
      | R.NotFound -> Ok None
      | R.Undefined _ -> Error (`Msg "invalid return")

  let write t path data =
    let module P = Api.Builder.Ctl.Write_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.path_set_list p path |> ignore;
    P.data_set p data;
    Capability.call_for_value t C.write_method req >|= function
    | Ok _ -> Ok ()
    | Error e -> errorf "error: write(%a) -> %a" pp_path path Capnp_rpc.Error.pp e

  let delete t path =
    let module P = Api.Builder.Ctl.Delete_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.path_set_list p path |> ignore;
    Capability.call_for_value t C.delete_method req >|= function
    | Ok _ -> Ok ()
    | Error e -> errorf "error: delete(%a) -> %a" pp_path path Capnp_rpc.Error.pp e

end

module Server = struct

  type op = [ `Read | `Write | `Delete ]

  type t = Api.Reader.Ctl.t Capability.t

  let infof fmt =
    Fmt.kstrf (fun msg () ->
        let date = Int64.of_float (Unix.gettimeofday ()) in
        Irmin.Info.v ~date ~author:"calf" msg
      ) fmt

  let not_allowed path =
    let err = Fmt.strf "%a is not an allowed path" pp_path path in
    Log.err (fun l -> l "%s" err);
    err

  let write db key value =
    let info = infof "Updating %a" KV.Key.pp key in
    KV.set db ~info key value

  let delete db key =
    let info = infof "Removing %a" KV.Key.pp key in
    KV.remove db ~info key

  let with_permission_check ~routes op key fn =
    match List.assoc key routes with
    | perms when List.mem op perms -> fn ()
    | _ -> Service.fail "%s" (not_allowed key)
    | exception Not_found -> Service.fail "%s" (not_allowed key)

  let service ~routes db =
    Api.Builder.Ctl.local @@
    object (_ : Api.Builder.Ctl.service)
      method read req =
        let module P = Api.Reader.Ctl.Read_params in
        let module R = Api.Builder.Response in
        let params = P.of_payload req in
        let key = P.path_get_list params in
        with_permission_check ~routes `Read key @@ fun () ->
        Service.return_lwt (fun () ->
            let resp, r = Service.Response.create R.init_pointer in
            (KV.find db key >|= function
              | None -> R.not_found_set r
              | Some x -> R.ok_set r x
            ) >|= fun () ->
            Ok resp
          )

      method write req =
        let module P = Api.Reader.Ctl.Write_params in
        let params = P.of_payload req in
        let key = P.path_get_list params in
        let value = P.data_get params in
        with_permission_check ~routes `Write key @@ fun () ->
        Service.return_lwt (fun () ->
            write db key value >|= fun () ->
            Ok (Service.Response.create_empty ())
          )

      method delete req =
        let module P = Api.Reader.Ctl.Delete_params in
        let params = P.of_payload req in
        let key = P.path_get_list params in
        with_permission_check ~routes `Delete key @@ fun () ->
        Service.return_lwt (fun () ->
            delete db key >|= fun () ->
            Ok (Service.Response.create_empty ())
          )
    end
end
