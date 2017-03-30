open Astring
open Lwt.Infix
open Sdk

let random_string n =
  Bytes.init n (fun _ -> char_of_int (Random.int 255))

(* workaround https://github.com/mirage/alcotest/issues/88 *)
exception Check_error of string

let check_raises msg exn f =
  Lwt.catch (fun () ->
      f () >>= fun () ->
      Lwt.fail (Check_error msg)
    ) (function
      | Check_error e    -> Alcotest.fail e
      | e ->
        if exn e then Lwt.return_unit
        else Fmt.kstrf Alcotest.fail "%s raised %a" msg Fmt.exn e)

let is_unix_error = function
  | Unix.Unix_error _ -> true
  | _ -> false

let escape = String.Ascii.escape

let write fd strs =
  Lwt_list.iter_s (fun str ->
      IO.really_write fd str 0 (String.length str)
    ) strs

let test_pipe pipe () =
  let calf = Init.Fd.fd @@ Init.Pipe.(calf pipe) in
  let priv = Init.Fd.fd @@ Init.Pipe.(priv pipe) in
  let name = Init.Pipe.name pipe in
  let test strs =
    let escape_strs = String.concat ~sep:"" @@ List.map escape strs in
    (* pipes are unidirectional *)
    (* calf -> priv works *)
    write calf strs >>= fun () ->
    IO.read_all priv >>= fun buf ->
    let msg = Fmt.strf "%s: calf -> priv" name in
    Alcotest.(check string) msg escape_strs (escape buf);
    (* priv -> calf don't *)
    check_raises (Fmt.strf "%s: priv side is writable!" name) is_unix_error
      (fun () -> write priv strs) >>= fun () ->
    check_raises (Fmt.strf "%s: calf sid is readable!" name) is_unix_error
      (fun () -> IO.read_all calf >|= ignore) >>= fun () ->
    Lwt.return_unit
  in
  test [random_string 1] >>= fun () ->
  test [random_string 1; random_string 1; random_string 10] >>= fun () ->
  test [random_string 100] >>= fun () ->
  test [random_string 10241] >>= fun () ->

  Lwt.return_unit

let test_socketpair pipe () =
  let calf = Init.Fd.fd @@ Init.Pipe.(calf pipe) in
  let priv = Init.Fd.fd @@ Init.Pipe.(priv pipe) in
  let name = Init.Pipe.name pipe in
  let test strs =
    let escape_strs = String.concat ~sep:"" @@ List.map escape strs in
    (* socket pairs are bi-directional *)
    (* calf -> priv works *)
    write calf strs >>= fun () ->
    IO.read_all priv >>= fun buf ->
    Alcotest.(check string) (name ^ " calf -> priv") escape_strs (escape buf);
    (* priv -> cal works *)
    write priv strs >>= fun () ->
    IO.read_all calf >>= fun buf ->
    Alcotest.(check string) (name ^ " priv -> calf") escape_strs (escape buf);
    Lwt.return_unit
  in
  test [random_string 1] >>= fun () ->
  test [random_string 1; random_string 1; random_string 10] >>= fun () ->
  test [random_string 100] >>= fun () ->
  (* note: if size(writes) > 8192 then the next writes will block (as
     we are using SOCK_STREAM *)
  let n = 8182 / 4 in
  test [
    random_string n;
    random_string n;
    random_string n;
    random_string n;
  ] >>= fun () ->

  Lwt.return_unit

let message = Alcotest.testable Ctl.Message.pp (=)

let test_message_serialization () =
  let test m =
    let buf = Ctl.Message.to_cstruct m in
    let m' = Ctl.Message.of_cstruct buf in
    Alcotest.(check message) "to_cstruct/of_cstruct" m m'
  in
  List.iter test [
    { operation = Read  ; path = "/foo/bar"; payload = ""    };
    { operation = Write ; path = ""        ; payload = "foo" };
    { operation = Delete; path = ""        ; payload = ""    };
    { operation = Delete; path = "foo"     ; payload = "foo" };
  ]

let test_message_send () =
  let calf = Init.Fd.fd @@ Init.Pipe.(calf ctl) in
  let priv = Init.Fd.fd @@ Init.Pipe.(priv ctl) in
  let test m =
    Ctl.Message.write calf m >>= fun () ->
    Ctl.Message.read priv >|= fun m' ->
    Alcotest.(check message) "write/read" m m'
  in
  Lwt_list.iter_s test [
    { operation = Read  ; path = "/foo/bar"; payload = ""    };
    { operation = Write ; path = ""        ; payload = "foo" };
    { operation = Delete; path = ""        ; payload = ""    };
    { operation = Delete; path = "foo"     ; payload = "foo" };
  ]

let run f () =
  try Lwt_main.run (f ())
  with e ->
    Fmt.epr "ERROR: %a" Fmt.exn e;
    raise e

let test_stderr () = ()

let test = [
  "stdout is a pipe"    , `Quick, run (test_pipe Init.Pipe.stdout);
  "stdout is a pipe"    , `Quick, run (test_pipe Init.Pipe.stderr);
  "net is a socket pair", `Quick, run (test_socketpair Init.Pipe.net);
  "ctl is a socket pair", `Quick, run (test_socketpair Init.Pipe.ctl);
  "seralize messages"   , `Quick, test_message_serialization;
  "send messages"       , `Quick, run test_message_send;
]

let reporter ?(prefix="") () =
  let pad n x =
    if String.length x > n then x
    else x ^ String.v ~len:(n - String.length x) (fun _ -> ' ')
  in
  let report src level ~over k msgf =
    let k _ = over (); k () in
    let ppf = match level with Logs.App -> Fmt.stdout | _ -> Fmt.stderr in
    let with_stamp h _tags k fmt =
      let dt = Mtime.to_us (Mtime.elapsed ()) in
      Fmt.kpf k ppf ("%s%+04.0fus %a %a @[" ^^ fmt ^^ "@]@.")
        prefix
        dt
        Fmt.(styled `Magenta string) (pad 10 @@ Logs.Src.name src)
        Logs_fmt.pp_header (level, h)
    in
    msgf @@ fun ?header ?tags fmt ->
    with_stamp header tags k fmt
  in
  { Logs.report = report }

let () =
  Logs.set_level (Some Logs.Debug);
  Logs.set_reporter (reporter ())

let () = Alcotest.run "sdk" [
    "init", test;
  ]
