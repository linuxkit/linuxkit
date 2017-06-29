open Lwt.Infix
open Common

let f ~socket ~share =
  connect socket >>= fun fd ->
  let oc = Lwt_io.of_fd ~mode:Lwt_io.Output fd in
  Lwt_io.write_line oc share >>= fun () ->
  Lwt_io.close oc
