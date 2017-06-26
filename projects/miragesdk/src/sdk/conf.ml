open Lwt.Infix
open Capnp_rpc_lwt

let src = Logs.Src.create "init" ~doc:"Init steps"
module Log = (val Logs.src_log src : Logs.LOG)

let pp_path = Fmt.(brackets (list ~sep:(const string "/") string))

let () =
  Irmin.Private.Watch.set_listen_dir_hook
    (fun _ _ _ -> Lwt.return (fun () -> Lwt.return_unit))
    (* FIXME: inotify need some unknown massaging. *)
    (* Irmin_watcher.hook *)

exception Undefined_field of int

let err_not_found fmt = Fmt.kstrf (fun x -> Lwt.fail_invalid_arg x) fmt
let failf fmt = Fmt.kstrf (fun x -> Lwt.fail_with x) fmt


module R = Api.Reader.Conf
module B = Api.Builder.Conf

module Callback = struct

  let service f =
    B.Callback.local @@ object (_: B.Callback.service)
      inherit B.Callback.service
      method f_impl req =
        let module P = R.Callback.F_params in
        let params = P.of_payload req in
        let change = P.change_get params in
        Service.return_lwt (fun () ->
            f change >|= fun () ->
            Ok (Service.Response.create_empty ())
          )
    end

  let client t change =
    let module P = B.Callback.F_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.change_set p change;
    Capability.call_for_value t R.Callback.f_method req >>= function
    | Ok _    -> Lwt.return ()
    | Error e -> failf "error: f(%s) -> %a" change Capnp_rpc.Error.pp e

end


module Client (F: Flow.S) = struct

  type t = R.t Capability.t

  let pp_error = Capnp_rpc.Error.pp

  let connect ~switch ?tags f =
    let ep = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) f in
    let client = Capnp_rpc_lwt.CapTP.connect ~switch ?tags ep in
    Capnp_rpc_lwt.CapTP.bootstrap client |> Lwt.return

  let find t path =
    let module P = B.Read_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.path_set_list p path |> ignore;
    Capability.call_for_value t R.read_method req >>= function
    | Error e -> failf "error read(%a): %a" pp_path path pp_error e
    | Ok r ->
      let module R = R.Response in
      let r = R.of_payload r in
      match R.get r with
      | R.Ok data     -> Lwt.return (Some data)
      | R.NotFound    -> Lwt.return None
      | R.Undefined _ -> failf "invalid return"

  let get t path =
    find t path >>= function
    | Some v -> Lwt.return v
    | None   -> err_not_found "get %a" pp_path path

  let set t path data =
    let module P = B.Write_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.path_set_list p path |> ignore;
    P.data_set p data;
    Capability.call_for_value t R.write_method req >>= function
    | Ok _    -> Lwt.return ()
    | Error e -> failf "error write(%a): %a" pp_path path pp_error e

  let delete t path =
    let module P = B.Delete_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.path_set_list p path |> ignore;
    Capability.call_for_value t R.delete_method req >>= function
    | Ok _    -> Lwt.return ()
    | Error e -> failf "error delete(%a): %a" pp_path path pp_error e

  let watch t path f =
    let module P = B.Watch_params in
    let req, p = Capability.Request.create P.init_pointer in
    P.path_set_list p path |> ignore;
    let callback = Capability.Request.export req (Callback.service f) in
    P.callback_set p (Some callback);
    Capability.call_for_value t R.watch_method req >>= function
    | Ok _    -> Lwt.return ()
    | Error e -> failf "error watch(%a): %a" pp_path path pp_error e

end

module Server (F: Flow.S) = struct

  module KV = struct

    module Store = Irmin_mem.KV

    include Store(Irmin.Contents.String)

    let v () =
      let config = Irmin_mem.config () in
      Repo.v config >>= fun repo ->
      of_branch repo "calf"

  end

  type op = [ `Read | `Write | `Delete ]

  type t = R.t Capability.t

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

  let contents_of_diff = function
    | `Added (_, `Contents (v, _))
    | `Updated (_, (_, `Contents (v, _))) -> Some v
    | _ -> None

  let watch ~switch db key f =
    KV.watch_key db key (fun diff ->
        match contents_of_diff diff with
        | Some v -> f v
        | None   -> Lwt.return ()
      ) >|= fun w ->
    Lwt_switch.add_hook (Some switch) (fun () -> KV.unwatch w)

  let with_permission_check ~routes op key fn =
    match List.assoc key routes with
    | perms when List.mem op perms -> fn ()
    | _ -> Service.fail "%s" (not_allowed key)
    | exception Not_found -> Service.fail "%s" (not_allowed key)

  let service ~switch ~routes db =
    B.local @@ object (_ : B.service)
      inherit B.service
      method read_impl req =
        let module P = R.Read_params in
        let params = P.of_payload req in
        let key = P.path_get_list params in
        with_permission_check ~routes `Read key @@ fun () ->
        Service.return_lwt (fun () ->
            let module R = B.Response in
            let resp, r = Service.Response.create R.init_pointer in
            (KV.find db key >|= function
              | None -> R.not_found_set r
              | Some x -> R.ok_set r x
            ) >|= fun () ->
            Ok resp
          )

      method write_impl req =
        let module P = R.Write_params in
        let params = P.of_payload req in
        let key = P.path_get_list params in
        let value = P.data_get params in
        with_permission_check ~routes `Write key @@ fun () ->
        Service.return_lwt (fun () ->
            write db key value >|= fun () ->
            Ok (Service.Response.create_empty ())
          )

      method delete_impl req =
        let module P = R.Delete_params in
        let params = P.of_payload req in
        let key = P.path_get_list params in
        with_permission_check ~routes `Delete key @@ fun () ->
        Service.return_lwt (fun () ->
            delete db key >|= fun () ->
            Ok (Service.Response.create_empty ())
          )

      method watch_impl req =
        let module P = R.Watch_params in
        let params = P.of_payload req in
        let key = P.path_get_list params in
        match P.callback_get params with
        | None   -> failwith "No watcher callback given"
        | Some i ->
          let callback = Payload.import req i in
          with_permission_check ~routes `Read key @@ fun () ->
          Service.return_lwt (fun () ->
              watch ~switch db key (Callback.client callback) >|= fun () ->
              Ok (Service.Response.create_empty ())
            )
    end

    let listen ~switch ?tags service fd =
      let endpoint = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) fd in
      Capnp_rpc_lwt.CapTP.connect ~switch ?tags ~offer:service endpoint
      |> ignore

end
