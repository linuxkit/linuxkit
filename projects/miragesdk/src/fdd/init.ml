open Lwt.Infix
open Common

let src = Logs.Src.create "fdd/init"
module Log = (val Logs.src_log src : Logs.LOG)

let write_pid socket =
  let pid_file = Filename.chop_extension socket ^ ".pid" in
  if Sys.file_exists pid_file then (
    Fmt.pr "Cannot start, as %s already exists.\n%!" pid_file;
    exit 1
  );
  Lwt_unix.openfile pid_file Lwt_unix.[O_CREAT; O_EXCL] 0o644 >>= fun fd ->
  Log.info (fun l -> l "Writing %s" pid_file);
  let oc = Lwt_io.of_fd ~mode:Lwt_io.Output fd in
  Lwt_io.write_line oc (string_of_int (Unix.getpid ())) >>= fun () ->
  Lwt_io.close oc >|= fun () ->
  at_exit (fun () ->
      Log.info (fun l -> l "Removing %s" pid_file);
      Unix.unlink pid_file
    )

(* listen on fd and send the socketpair to the first 2 connections.*)
let send_socketpair fd =
  let f, d = Lwt_unix.(socketpair PF_UNIX SOCK_STREAM 0) in
  let send to_send =
    Lwt_unix.accept fd >>= fun (fd, _) ->
    Log.info (fun l -> l "New client!");
    send_fd ~to_send fd
  in
  Lwt_unix.listen fd 2;
  Lwt.join [send f; send d]

let recv_path fd =
  let ic = Lwt_io.of_fd ~mode:Lwt_io.Input fd in
  Lwt_io.read_line ic >>= fun line ->
  let path = String.trim line in
  bind path >>= fun fd ->
  send_socketpair fd >>= fun () ->
  Lwt_unix.unlink path

let listen fd =
  let rec loop () =
    Lwt_unix.accept fd >>= fun (fd, _) ->
    Log.debug (fun l -> l "New client connected!");
    Lwt.async (fun () ->
        Lwt.catch
          (fun () -> recv_path fd)
          (fun e  ->
             Log.err (fun l -> l "asynchronous exn: %a" Fmt.exn e);
             Lwt.return ())
      );
    loop ()
  in
  Lwt_unix.listen fd 10;
  loop ()

let f socket =
  write_pid socket >>= fun () ->
  bind socket >>= fun fd ->
  listen fd
