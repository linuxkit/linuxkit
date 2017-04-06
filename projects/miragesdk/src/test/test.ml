open Astring
open Lwt.Infix
open Sdk

let random_string n =
  Bytes.init n (fun _ -> char_of_int (Random.int 255))

(* workaround https://github.com/mirage/alcotest/issues/88 *)
exception Check_error of string

let check_raises msg f =
  Lwt.catch (fun () ->
      f () >>= fun () ->
      Lwt.fail (Check_error msg)
    ) (function
      | Check_error e -> Alcotest.fail e
      | _             -> Lwt.return_unit
    )

let escape = String.Ascii.escape

let write fd strs =
  Lwt_list.iter_s (fun str ->
      IO.write fd (Cstruct.of_string str) >>= function
      | Ok ()   -> Lwt.return_unit
      | Error e -> Fmt.kstrf Lwt.fail_with "write: %a" IO.pp_write_error e
    ) strs

let read fd =
  IO.read fd >>= function
  | Ok (`Data x) -> Lwt.return (Cstruct.to_string x)
  | Ok `Eof      -> Lwt.fail_with "read: EOF"
  | Error e      -> Fmt.kstrf Lwt.fail_with "read: %a" IO.pp_error e

let calf pipe = Init.(Fd.flow Pipe.(calf pipe))
let priv pipe = Init.(Fd.flow Pipe.(priv pipe))

let test_pipe pipe () =
  let calf = calf pipe in
  let priv = priv pipe in
  let name = Init.Pipe.name pipe in
  let test strs =
    let escape_strs = String.concat ~sep:"" @@ List.map escape strs in
    (* pipes are unidirectional *)
    (* calf -> priv works *)
    write calf strs >>= fun () ->
    read priv >>= fun buf ->
    let msg = Fmt.strf "%s: calf -> priv" name in
    Alcotest.(check string) msg escape_strs (escape buf);
    (* priv -> calf don't *)
    check_raises (Fmt.strf "%s: priv side is writable!" name)
      (fun () -> write priv strs) >>= fun () ->
    check_raises (Fmt.strf "%s: calf sid is readable!" name)
      (fun () -> read calf >|= ignore) >>= fun () ->
    Lwt.return_unit
  in
  test [random_string 1] >>= fun () ->
  test [random_string 1; random_string 1; random_string 10] >>= fun () ->
  test [random_string 100] >>= fun () ->
  test [random_string 10241] >>= fun () ->

  Lwt.return_unit

let test_socketpair pipe () =
  let calf = calf pipe in
  let priv = priv pipe in
  let name = Init.Pipe.name pipe in
  let test strs =
    let escape_strs = String.concat ~sep:"" @@ List.map escape strs in
    (* socket pairs are bi-directional *)
    (* calf -> priv works *)
    write calf strs >>= fun () ->
    read priv >>= fun buf ->
    Alcotest.(check string) (name ^ " calf -> priv") escape_strs (escape buf);
    (* priv -> cal works *)
    write priv strs >>= fun () ->
    read calf >>= fun buf ->
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

let test_send t write read message pp_error messages =
  let calf = calf Init.Pipe.(ctl t) in
  let priv = priv Init.Pipe.(ctl t) in
  let test m =
    write calf m >>= fun () ->
    read priv >|= function
    | Ok m'   -> Alcotest.(check message) "write/read" m m'
    | Error e -> Fmt.kstrf Alcotest.fail "Message.read: %a" pp_error e
  in
  Lwt_list.iter_s test messages

let test_query_serialization () =
  let open Ctl.Query in
  test_serialization to_cstruct of_cstruct query queries

let test_reply_serialization () =
  let open Ctl.Reply in
  test_serialization to_cstruct of_cstruct reply replies

let test_query_send t () =
  let open Ctl.Query in
  test_send t write read query pp_error queries

let test_reply_send t () =
  let open Ctl.Reply in
  test_send t write read reply pp_error replies

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

let test_ctl t () =
  let calf = calf Init.Pipe.(ctl t) in
  let priv = priv Init.Pipe.(ctl t) in
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

let in_memory_flow () =
  let flow = Mirage_flow_lwt.F.string () in
  IO.create (module Mirage_flow_lwt.F) flow "mem"

let test_exec () =
  let test () =
    let check n pipe =
      let t = Init.Pipe.v () in
      let pipe = pipe t in
      Init.exec t ["/bin/sh"; "-c"; "echo foo >& " ^ string_of_int n] @@ fun _pid ->
      read @@ priv pipe >>= fun foo ->
      let name = Fmt.strf "fork %s" Init.Pipe.(name pipe) in
      Alcotest.(check string) name "foo\n" foo;
      Lwt.return_unit
    in
    check 1 Init.Pipe.stdout >>= fun () ->
    (* avoid logging interference *)
    let level = Logs.level () in
    Logs.set_level None;
    check 2 Init.Pipe.stderr >>= fun () ->
    Logs.set_level level;
    check 3 Init.Pipe.net    >>= fun () ->
    check 4 Init.Pipe.ctl    >>= fun () ->
    Lwt.return_unit
  in
  test ()

let run f () =
  try Lwt_main.run (f ())
  with e ->
    Fmt.epr "ERROR: %a" Fmt.exn e;
    raise e

let test_stderr () = ()

let t = Init.Pipe.v ()

let test = [
  "stdout is a pipe"    , `Quick, run (test_pipe Init.Pipe.(stdout t));
  "stdout is a pipe"    , `Quick, run (test_pipe Init.Pipe.(stderr t));
  "net is a socket pair", `Quick, run (test_socketpair Init.Pipe.(net t));
  "ctl is a socket pair", `Quick, run (test_socketpair Init.Pipe.(ctl t));
  "seralize queries"    , `Quick, test_query_serialization;
  "seralize replies"    , `Quick, test_reply_serialization;
  "send queries"        , `Quick, run (test_query_send t);
  "send replies"        , `Quick, run (test_reply_send t);
  "ctl"                 , `Quick, run (test_ctl t);
  "exec"                , `Quick, run test_exec;
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
