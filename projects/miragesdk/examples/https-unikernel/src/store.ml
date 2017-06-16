let src = Logs.Src.create "web.store" ~doc:"Datastore for web server"
module Log = (val Logs.src_log src: Logs.LOG)

module Irmin_store = Irmin_unix.Git.FS.KV(Irmin.Contents.String)

open Lwt.Infix
open Capnp_rpc_lwt
open Astring

(* The Cap'n'Proto service interface we expose. *)
let service () =
  let config = Irmin_fs.config "www-data" in 
  Irmin_store.Repo.v config >>= fun repo ->
  Irmin_store.of_branch repo Irmin_store.Branch.master >|= fun db ->
  Api.Builder.Store.local @@
    object (_ : Api.Builder.Store.service)
      method get req =
        let module P = Api.Reader.Store.Get_params in
        let module R = Api.Builder.Store.GetResults in
        let params = P.of_payload req in
        let path = P.path_get_list params in
        Log.info (fun f -> f "Handing request for %a" (Fmt.Dump.list String.dump) path);
        Service.return_lwt (fun () ->
          let resp, results = Service.Response.create R.init_pointer in
          begin
            Irmin_store.find db path >|= function
            | Some data -> R.ok_set results data
            | None -> R.not_found_set results
          end
          >>= fun () ->
          Lwt.return (Ok resp)
        )
    end
