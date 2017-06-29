open Lwt.Infix

let src = Logs.Src.create "fdd/common"
module Log = (val Logs.src_log src : Logs.LOG)

let magic_header = "FDD0"

let bind path =
  Log.debug (fun l -> l "bind %s" path);
  Lwt.catch (fun () -> Lwt_unix.unlink path) (fun _ -> Lwt.return ())
  >>= fun () ->
  let fd = Lwt_unix.socket Lwt_unix.PF_UNIX Lwt_unix.SOCK_STREAM 0 in
  Lwt_unix.bind fd (Lwt_unix.ADDR_UNIX path) >|= fun () ->
  fd

let connect path =
  Log.debug (fun l -> l "connect %s" path);
  let fd = Lwt_unix.socket Lwt_unix.PF_UNIX Lwt_unix.SOCK_STREAM 0 in
  Lwt_unix.connect fd (Lwt_unix.ADDR_UNIX path) >|= fun () ->
  fd

let send_fd ~to_send fd =
  Log.debug (fun l -> l "send_fd");
  let fd = Lwt_unix.unix_file_descr fd in
  let to_send = Lwt_unix.unix_file_descr to_send in
  let len = String.length magic_header in
  Lwt_preemptive.detach (fun () ->
      let i = Fd_send_recv.send_fd fd magic_header 0 len [] to_send in
      assert (i = len)
    ) ()

let recv_fd fd =
  Log.debug (fun l -> l "recv_fd");
  let len = String.length magic_header in
  let buf = Bytes.create len in
  let fd = Lwt_unix.unix_file_descr fd in
  Lwt_preemptive.detach (fun () ->
      Unix.clear_nonblock fd;
      Fd_send_recv.recv_fd fd buf 0 len []
    ) ()
  >|= fun (n, _, c) ->
  Log.debug (fun l -> l "recv_fd: received %S (%d)" buf n);
  assert (n = len);
  assert (buf = magic_header);
  Lwt_unix.of_unix_file_descr c
