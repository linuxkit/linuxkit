(** Run the Store service as a stand-alone Unix process. *)

open Lwt.Infix

let () = Logging.init ()

let main store_socket =
  Lwt_main.run begin
    Store.local () >>= fun store ->
    Capnp_rpc_unix.serve ~offer:store store_socket
  end

open Cmdliner

let store =
  let doc = "The database store to serve" in
  Arg.(required @@ pos 0 (some Capnp_rpc_unix.Listen_address.conv) None @@ info ~doc ~docv:"STORE" [])

let cmd =
  Term.(const main $ store), Term.info "store" 

let () = Term.(exit @@ eval cmd)
