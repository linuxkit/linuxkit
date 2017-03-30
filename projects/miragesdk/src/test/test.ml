open Astring
open Lwt.Infix
open Sdk

let random_string n = Bytes.create n

let test_pipe pipe () =
  let calf = Init.Fd.fd @@ Init.Pipe.(calf pipe) in
  let priv = Init.Fd.fd @@ Init.Pipe.(priv pipe) in
  let test str =
    (* check the the pipe is unidirectional *)
    IO.really_write calf str 0 (String.length str) >>= fun () ->
    IO.read_all priv >>= fun buf ->
    Alcotest.(check string) "stdout"
      (String.Ascii.escape str) (String.Ascii.escape buf);
    Lwt.catch (fun () ->
        IO.really_write priv str 0 (String.length str) >|= fun () ->
        Alcotest.fail "priv side is writable!"
      ) (fun _ -> Lwt.return_unit)
    >>= fun () ->
    Lwt.catch (fun () ->
        IO.read_all calf >|= fun _ ->
        Alcotest.fail "calf sid is readable!"
      ) (fun _ -> Lwt.return_unit)
    >>= fun () ->
    Lwt.return_unit
  in
  test (random_string 1) >>= fun () ->
  test (random_string 100) >>= fun () ->
  test (random_string 10241) >>= fun () ->

  Lwt.return_unit

let run f () =
  try Lwt_main.run (f ())
  with e ->
    Fmt.epr "ERROR: %a" Fmt.exn e;
    raise e

let test_stderr () = ()

let test = [
  "stdout" , `Quick, run (test_pipe Init.Pipe.stdout);
  "stdout" , `Quick, run (test_pipe Init.Pipe.stderr);
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
