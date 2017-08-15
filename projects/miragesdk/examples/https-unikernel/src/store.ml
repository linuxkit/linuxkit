(** The implementation of the store.
    This just looks up the requested page in an Irmin database. *)

let src = Logs.Src.create "web.store" ~doc:"Datastore for web server"
module Log = (val Logs.src_log src: Logs.LOG)

module Irmin_store = Irmin_unix.Git.FS.KV(Irmin.Contents.String)

open Lwt.Infix
open Astring

let local () =
  let config = Irmin_fs.config "www-data" in 
  Irmin_store.Repo.v config >>= fun repo ->
  Irmin_store.of_branch repo Irmin_store.Branch.master >|= fun db ->
  Rpc.Store.local (fun path ->
      Log.info (fun f -> f "Handing request for %a" (Fmt.Dump.list String.dump) path);
      Irmin_store.find db path
    )
