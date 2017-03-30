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

let query = Alcotest.testable Ctl.Query.pp (=)
let reply = Alcotest.testable Ctl.Reply.pp (=)

let queries =
  let open Ctl.Query in
  [
    { version = 0l; id = 0l; operation = Read; path = "/foo/bar"; payload = "" };
    { version = Int32.max_int; id = Int32.max_int; operation = Write ; path = ""; payload = "foo" };
    { version = 1l;id = 0l; operation = Delete; path = ""; payload = "" };
    { version = -2l; id = -3l; operation = Delete; path = "foo"; payload = "foo" };
  ]

let replies =
  let open Ctl.Reply in
  [
    { id = 0l; status = Ok; payload = "" };
    { id = Int32.max_int; status = Ok; payload = "foo" };
    { id = 0l; status = Error; payload = "" };
    { id = -3l; status = Error; payload = "foo" };
  ]

let test_serialization to_cstruct of_cstruct message messages =
  let test m =
    let buf = to_cstruct m in
    match of_cstruct buf with
    | Ok m' -> Alcotest.(check message) "to_cstruct/of_cstruct" m m'
    | Error (`Msg e) -> Alcotest.fail ("Message.of_cstruct: " ^ e)
  in
  List.iter test messages

let test_send write read message messages =
  let calf = Init.Fd.fd @@ Init.Pipe.(calf ctl) in
  let priv = Init.Fd.fd @@ Init.Pipe.(priv ctl) in
  let test m =
    write calf m >>= fun () ->
    read priv >|= function
    | Ok m' -> Alcotest.(check message) "write/read" m m'
    | Error (`Msg e) -> Alcotest.fail ("Message.read: " ^ e)
  in
  Lwt_list.iter_s test messages

let test_query_serialization () =
  let open Ctl.Query in
  test_serialization to_cstruct of_cstruct query queries

let test_reply_serialization () =
  let open Ctl.Reply in
  test_serialization to_cstruct of_cstruct reply replies

let test_query_send () =
  let open Ctl.Query in
  test_send write read query queries

let test_reply_send () =
  let open Ctl.Reply in
  test_send write read reply replies

let failf fmt = Fmt.kstrf Alcotest.fail fmt

(* read ops *)

let read_should_err t k =
  Ctl.Client.read t k >|= function
  | Error (`Msg _) -> ()
  | Ok None        -> failf "read(%s) -> got: none, expected: err" k
  | Ok Some v      -> failf "read(%s) -> got: found:%S, expected: err" k v

let read_should_none t k =
  Ctl.Client.read t k >|= function
  | Error (`Msg e) -> failf "read(%s) -> got: error:%s, expected none" k e
  | Ok None        -> ()
  | Ok Some v      -> failf "read(%s) -> got: found:%S, expected none" k v

let read_should_work t k v =
  Ctl.Client.read t k >|= function
  | Error (`Msg e) -> failf "read(%s) -> got: error:%s, expected ok" k e
  | Ok None        -> failf "read(%s) -> got: none, expected ok" k
  | Ok Some v'     ->
    if v <> v' then failf "read(%s) -> got: ok:%S, expected: ok:%S" k v' v

(* write ops *)

let write_should_err t k v =
  Ctl.Client.write t k v >|= function
  | Ok ()   -> failf "write(%s) -> ok" k
  | Error _ -> ()

let write_should_work t k v =
  Ctl.Client.write t k v >|= function
  | Ok ()          -> ()
  | Error (`Msg e) -> failf "write(%s) -> error: %s" k e

(* del ops *)

let delete_should_err t k =
  Ctl.Client.delete t k >|= function
  | Ok ()   -> failf "del(%s) -> ok" k
  | Error _ -> ()

let delete_should_work t k =
  Ctl.Client.delete t k >|= function
  | Ok ()          -> ()
  | Error (`Msg e) -> failf "write(%s) -> error: %s" k e

let test_ctl () =
  let calf = Init.Fd.fd @@ Init.Pipe.(calf ctl) in
  let priv = Init.Fd.fd @@ Init.Pipe.(priv ctl) in
  let k1 = "/foo/bar" in
  let k2 = "a" in
  let k3 = "b/c" in
  let k4 = "xxxxxx" in
  let routes = [k1; k2; k3] in
  let git_root = "/tmp/sdk/ctl" in
  let _ = Sys.command (Fmt.strf "rm -rf %s" git_root) in
  Ctl.v git_root >>= fun ctl ->
  let server () = Ctl.Server.listen ~routes ctl priv in
  let client () =
    let t = Ctl.Client.v calf in
    let allowed k v =
      delete_should_work t k >>= fun () ->
      read_should_none t k    >>= fun () ->
      write_should_work t k v >>= fun () ->
      read_should_work t k v  >>= fun () ->
      let path = String.cuts ~empty:false ~sep:"/" k in
      Ctl.KV.get ctl path     >|= fun v' ->
      Alcotest.(check string) "in the db" v v'
    in
    let disallowed k v =
      read_should_err t k    >>= fun () ->
      write_should_err t k v >>= fun () ->
      delete_should_err t k
    in
    allowed k1 ""                           >>= fun () ->
    allowed k2 "xxx"                        >>= fun () ->
    allowed k3 (random_string (255 * 1024)) >>= fun () ->
    disallowed k4 "" >>= fun () ->
    Lwt.return_unit
  in
  Lwt.pick [
    client ();
    server ();
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
  "seralize queries"    , `Quick, test_query_serialization;
  "seralize replies"    , `Quick, test_reply_serialization;
  "send queries"        , `Quick, run test_query_send;
  "send replies"        , `Quick, run test_reply_send;
  "ctl"                 , `Quick, run test_ctl;
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
