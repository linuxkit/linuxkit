open Lwt.Infix
open Common

let src = Logs.Src.create "fdd/share"
module Log = (val Logs.src_log src : Logs.LOG)

let sleep ?(sleep_t=0.01) () =
  let sleep_t = min sleep_t 1. in
  Lwt_unix.yield () >>= fun () ->
  Lwt_unix.sleep sleep_t

let retry ?(timeout=5. *. 60.) ?(sleep_t=0.) fn =
  let sleep_t = max sleep_t 0.001 in
  let time = Unix.gettimeofday in
  let t = time () in
  let str i = Fmt.strf "%d, %.3fs" i (time () -. t) in
  let rec aux i =
    if time () -. t > timeout then fn ()
    else
      Lwt.catch fn (fun ex ->
          Log.debug (fun f -> f "retry ex: %a" Fmt.exn ex);
          let sleep_t = sleep_t *. (1. +. float i ** 2.) in
          sleep ~sleep_t () >>= fun () ->
          Log.debug (fun f -> f "Test.retry %s" (str i));
          aux (i+1)
        )
  in
  aux 0

let f ~socket ~share =
  retry (fun () -> connect socket) >>= fun fd ->
  let oc = Lwt_io.of_fd ~mode:Lwt_io.Output fd in
  Lwt_io.write_line oc share >>= fun () ->
  Lwt_io.close oc
