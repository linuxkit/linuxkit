open Lwt.Infix
open Common

let src = Logs.Src.create "fdd/exec"
module Log = (val Logs.src_log src : Logs.LOG)

let get_fd share = connect share >>= recv_fd

let fd_of_int (i:int) =
  let fd : Unix.file_descr = Obj.magic i in
  Lwt_unix.of_unix_file_descr fd

let dup (share, fds) =
  Log.info (fun l ->
      l "mapping %s to fds: %a" share Fmt.(list ~sep:(unit " ") int) fds);
  get_fd share >|= fun fd ->
  List.iter (fun n -> Lwt_unix.dup2 fd (fd_of_int n)) fds;
  Unix.close (Lwt_unix.unix_file_descr fd)

let f dups cmd =
  Lwt_list.iter_p dup dups >>= fun () ->
  Unix.execvp (List.hd cmd) (Array.of_list cmd)
