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

module Callback = struct

  let service f =
    let open Api.Service.Conf.Callback in
    local @@ object (_: service)
      inherit service
      method f_impl req release_param_caps =
        let change = F.Params.change_get req in
        release_param_caps ();
        Service.return_lwt (fun () ->
            f change >|= fun () ->
            Ok (Service.Response.create_empty ())
          )
    end

  let client t change =
    let open Api.Client.Conf.Callback in
    let req, p = Capability.Request.create F.Params.init_pointer in
    F.Params.change_set p change;
    Capability.call_for_value_exn t F.method_id req >|=
    ignore

end


module Client (F: Flow.S) = struct

  module Conf = Api.Client.Conf

  type t = Conf.t Capability.t

  let connect ~switch ?tags f =
    let ep = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) f in
    let client = Capnp_rpc_lwt.CapTP.connect ~switch ?tags ep in
    Capnp_rpc_lwt.CapTP.bootstrap client |> Lwt.return

  let find t path =
    let open Conf.Read in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.path_set_list p path |> ignore;
    Capability.call_for_value_exn t method_id req >>= fun r ->
    match Results.get r with
    | Ok data     -> Lwt.return (Some data)
    | NotFound    -> Lwt.return None
    | Undefined _ -> failf "invalid return"

  let get t path =
    find t path >>= function
    | Some v -> Lwt.return v
    | None   -> err_not_found "get %a" pp_path path

  let set t path data =
    let open Conf.Write in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.path_set_list p path |> ignore;
    Params.data_set p data;
    Capability.call_for_value_exn t method_id req >|= ignore

  let delete t path =
    let open Conf.Delete in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.path_set_list p path |> ignore;
    Capability.call_for_value_exn t method_id req >|= ignore

  let watch t path f =
    let open Conf.Watch in
    let req, p = Capability.Request.create Params.init_pointer in
    Params.path_set_list p path |> ignore;
    Params.callback_set p (Some (Callback.service f));
    Capability.call_for_value_exn t method_id req >|= ignore

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

  module Conf = Api.Service.Conf
  type t = Conf.t Capability.t

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
    Conf.local @@ object (_ : Conf.service)
      inherit Conf.service
      method read_impl req release_param_caps =
        let open Conf.Read in
        let key = Params.path_get_list req in
        release_param_caps ();
        with_permission_check ~routes `Read key @@ fun () ->
        Service.return_lwt (fun () ->
            let resp, r = Service.Response.create Results.init_pointer in
            (KV.find db key >|= function
              | None   -> Results.not_found_set r
              | Some x -> Results.ok_set r x
            ) >|= fun () ->
            Ok resp
          )

      method write_impl req release_param_caps =
        let open Conf.Write in
        let key = Params.path_get_list req in
        let value = Params.data_get req in
        release_param_caps ();
        with_permission_check ~routes `Write key @@ fun () ->
        Service.return_lwt (fun () ->
            write db key value >|= fun () ->
            Ok (Service.Response.create_empty ())
          )

      method delete_impl req release_param_caps =
        let open Conf.Delete in
        let key = Params.path_get_list req in
        release_param_caps ();
        with_permission_check ~routes `Delete key @@ fun () ->
        Service.return_lwt (fun () ->
            delete db key >|= fun () ->
            Ok (Service.Response.create_empty ())
          )

      method watch_impl req release_param_caps =
        let open Conf.Watch in
        let key = Params.path_get_list req in
        let callback = Params.callback_get req in
        release_param_caps ();
        match callback with
        | None   -> Service.fail "No watcher callback given"
        | Some i ->
          with_permission_check ~routes `Read key @@ fun () ->
          Service.return_lwt (fun () ->
              watch ~switch db key (Callback.client i) >|= fun () ->
              Ok (Service.Response.create_empty ())
            )
    end

    let listen ~switch ?tags service fd =
      let endpoint = Capnp_rpc_lwt.Endpoint.of_flow ~switch (module F) fd in
      Capnp_rpc_lwt.CapTP.connect ~switch ?tags ~offer:service endpoint
      |> ignore

end
