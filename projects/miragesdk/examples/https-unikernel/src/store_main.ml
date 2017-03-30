open Lwt.Infix

let () = Common.init_logging ()

let main store_socket =
  Lwt_main.run begin
    Lwt_switch.with_switch @@ fun switch ->
    Store.service () >>= fun store ->
    Common.listen ~switch ~offer:store store_socket
  end

open Cmdliner

let store =
  let doc = "The database store to serve" in
  Arg.(required @@ pos 0 (some string) None @@ info ~doc ~docv:"STORE" [])

let cmd =
  Term.(const main $ store), Term.info "store" 

let () = Term.(exit @@ eval cmd)
