open Lwt.Infix

let src = Logs.Src.create "IO" ~doc:"IO helpers"
module Log = (val Logs.src_log src : Logs.LOG)

let rec really_write fd buf off len =
  Log.debug (fun l -> l "really_write");
  match len with
  | 0   -> Lwt.return_unit
  | len ->
    Lwt_unix.write fd buf off len >>= fun n ->
    really_write fd buf (off+n) (len-n)

let rec really_read fd buf off len =
  Log.debug (fun l -> l "really_read");
  match len with
  | 0   -> Lwt.return_unit
  | len ->
    Lwt_unix.read fd buf off len >>= fun n ->
    really_read fd buf (off+n) (len-n)

let read_all fd =
  Log.debug (fun l -> l "read_all");
  let len = 16 * 1024 in
  let buf = Bytes.create len in
  let rec loop acc =
    Lwt_unix.read fd buf 0 len >>= fun n ->
    let acc = String.sub buf 0 n :: acc in
    if n <= len then Lwt.return (List.rev acc)
    else loop acc
  in
  loop [] >|= fun bufs ->
  String.concat "" bufs

let read_n fd len =
  Log.debug (fun l -> l "read_n");
  let buf = Bytes.create len in
  let rec loop acc len =
    Lwt_unix.read fd buf 0 len >>= fun n ->
    let acc = String.sub buf 0 n :: acc in
    match len - n with
    | 0 -> Lwt.return (List.rev acc)
    | r -> loop acc r
  in
  loop [] len >|= fun bufs ->
  String.concat "" bufs
