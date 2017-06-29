open Lwt.Infix
open Common

let get_fd share = connect share >>= recv_fd

let red = Fmt.(styled `Red string)
let green = Fmt.(styled `Green string)

let f share =
  if not (Sys.file_exists share) then (
    Fmt.pr "%a %s does not exist.\n%!" red "[ERROR]" share;
    exit 1;
  );
  get_fd share >>= fun x ->
  get_fd share >>= fun y ->
  let x = Lwt_io.of_fd ~mode:Lwt_io.Output x in
  let y = Lwt_io.of_fd ~mode:Lwt_io.Input y in
  let payload = "This is a test!" in
  Lwt_io.write_line x payload >>= fun () ->
  Lwt_io.read_line y >|= fun buf ->
  if buf <> payload then (
    Fmt.pr "%a Expecting %S, but got %S.\n%!" red "[ERROR]" payload buf;
    exit 1
  ) else (
    Fmt.pr "%a the socketpair which was shared on %s is working properly.\n%!"
      green "[SUCCES]" share
  )
