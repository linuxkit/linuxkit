open Lwt.Infix

let src = Logs.Src.create "IO" ~doc:"IO helpers"
module Log = (val Logs.src_log src : Logs.LOG)

let rec really_write fd buf off len =
  match len with
  | 0   -> Lwt.return_unit
  | len ->
    Log.debug (fun l -> l "really_write off=%d len=%d" off len);
    Lwt_unix.write fd buf off len >>= fun n ->
    if n = 0 then Lwt.fail_with "write 0"
    else really_write fd buf (off+n) (len-n)

let write fd buf = really_write fd buf 0 (String.length buf)

let rec really_read fd buf off len =
  match len with
  | 0   -> Lwt.return_unit
  | len ->
    Log.debug (fun l -> l "really_read off=%d len=%d" off len);
    Lwt_unix.read fd buf off len >>= fun n ->
    if n = 0 then Lwt.fail_with "read 0"
    else really_read fd buf (off+n) (len-n)

let read_all fd =
  Log.debug (fun l -> l "read_all");
  let len = 16 * 1024 in
  let buf = Bytes.create len in
  let rec loop acc =
    Lwt_unix.read fd buf 0 len >>= fun n ->
    if n = 0 then Lwt.fail_with "read 0"
    else
      let acc = String.sub buf 0 n :: acc in
      if n <= len then Lwt.return (List.rev acc)
      else loop acc
  in
  loop [] >|= fun bufs ->
  String.concat "" bufs

let read_n fd len =
  Log.debug (fun l -> l "read_n len=%d" len);
  let buf = Bytes.create len in
  let rec loop acc len =
    Lwt_unix.read fd buf 0 len >>= fun n ->
    if n = 0 then Lwt.fail_with "read 0"
    else
      let acc = String.sub buf 0 n :: acc in
      match len - n with
      | 0 -> Lwt.return (List.rev acc)
      | r -> loop acc r
  in
  loop [] len >|= fun bufs ->
  String.concat "" bufs
