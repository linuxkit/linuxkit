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

let request = Alcotest.testable Ctl.Request.pp Ctl.Request.equal
let response = Alcotest.testable Ctl.Response.pp Ctl.Response.equal

let queries =
  let open Ctl.Request in
  [
    v ~id:0l ~path:["foo";"bar"] Read;
    v ~id:Int32.max_int ~path:[] (Write "foo");
    v ~id:0l ~path:[] Delete;
    v ~id:(-3l) ~path:["foo"] Delete;
  ]

let replies =
  let open Ctl.Response in
  [
    v ~id:0l (Ok "");
    v ~id:Int32.max_int (Ok "foo");
    v ~id:0l (Error "");
    v ~id:(-3l) (Error "foo");
  ]

let failf fmt = Fmt.kstrf Alcotest.fail fmt

let test_send t write read message messages =
  let calf = Ctl.Endpoint.v @@ calf Init.Pipe.(ctl t) in
  let priv = Ctl.Endpoint.v @@ priv Init.Pipe.(ctl t) in
  let test m =
    write calf m >>= function
    | Error e -> failf "Message.write: %a" Ctl.Endpoint.pp_error e
    | Ok ()   ->
      read priv >|= function
      | Ok m'   -> Alcotest.(check message) "write/read" m m'
      | Error e -> failf "Message.read: %a" Ctl.Endpoint.pp_error e
  in
  Lwt_list.iter_s test messages

let test_request_send t () =
  let open Ctl.Request in
  test_send t write read request queries

let test_response_send t () =
  let open Ctl.Response in
  test_send t write read response replies

let failf fmt = Fmt.kstrf Alcotest.fail fmt

(* read ops *)

let pp_error = Ctl.Client.pp_error
let pp_path = Fmt.(Dump.list string)

let read_should_err t k =
  Ctl.Client.read t k >|= function
  | Error _   -> ()
  | Ok None   -> failf "read(%a) -> got: none, expected: err" pp_path k
  | Ok Some v -> failf "read(%a) -> got: found:%S, expected: err" pp_path k v

let read_should_none t k =
  Ctl.Client.read t k >|= function
  | Error e   -> failf "read(%a) -> got: error:%a, expected none" pp_path k pp_error e
  | Ok None   -> ()
  | Ok Some v -> failf "read(%a) -> got: found:%S, expected none" pp_path k v

let read_should_work t k v =
  Ctl.Client.read t k >|= function
  | Error e    -> failf "read(%a) -> got: error:%a, expected ok" pp_path k pp_error e
  | Ok None    -> failf "read(%a) -> got: none, expected ok" pp_path k
  | Ok Some v' ->
    if v <> v' then failf "read(%a) -> got: ok:%S, expected: ok:%S" pp_path k v' v

(* write ops *)

let write_should_err t k v =
  Ctl.Client.write t k v >|= function
  | Ok ()   -> failf "write(%a) -> ok" pp_path k
  | Error _ -> ()

let write_should_work t k v =
  Ctl.Client.write t k v >|= function
  | Ok ()   -> ()
  | Error e -> failf "write(%a) -> error: %a" pp_path k pp_error e

(* del ops *)

let delete_should_err t k =
  Ctl.Client.delete t k >|= function
  | Ok ()   -> failf "del(%a) -> ok" pp_path k
  | Error _ -> ()

let delete_should_work t k =
  Ctl.Client.delete t k >|= function
  | Ok ()   -> ()
  | Error e -> failf "write(%a) -> error: %a" pp_path k pp_error e

let test_ctl t () =
  let calf = calf Init.Pipe.(ctl t) in
  let priv = priv Init.Pipe.(ctl t) in
  let k1 = ["foo"; "bar"] in
  let k2 = ["a"] in
  let k3 = ["b"; "c"] in
  let k4 = ["xxxxxx"] in
  let all = [`Read; `Write; `Delete] in
  let routes = [k1,all; k2,all; k3,all ] in
  let git_root = "/tmp/sdk/ctl" in
  let _ = Sys.command (Fmt.strf "rm -rf %s" git_root) in
  Ctl.v git_root >>= fun ctl ->
  let server () = Ctl.Server.listen ~routes ctl priv in
  let client () =
    let t = Ctl.Client.v calf in
    let allowed k v =
      delete_should_work t k  >>= fun () ->
      read_should_none t k    >>= fun () ->
      write_should_work t k v >>= fun () ->
      read_should_work t k v  >>= fun () ->
      Ctl.KV.get ctl k        >|= fun v' ->
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
  "send requests"       , `Quick, run (test_request_send t);
  "send responses"      , `Quick, run (test_response_send t);
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
