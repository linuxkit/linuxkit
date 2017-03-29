open Lwt.Infix
open Sdk

let random_string n = Bytes.create n

let test_pipe pipe () =
  let ic = Init.Fd.fd @@ Init.Pipe.(calf pipe) in
  let oc = Init.Fd.fd @@ Init.Pipe.(priv pipe) in
  let test str =
    Init.IO.really_write oc str 0 (String.length str) >>= fun () ->
    Init.IO.read_all ic >|= fun buf ->
    Alcotest.(check string) "stdout" str buf
  in
  test (random_string 10241) >>= fun () ->
  test (random_string 100) >>= fun () ->
  test (random_string 1)

let run f () =
  try Lwt_main.run (f ())
  with e -> Fmt.epr "ERROR: %a" Fmt.exn e

let test_stderr () = ()

let test = [
  "stdout", `Quick, run (test_pipe Init.Pipe.stdout);
  "stdout", `Quick, run (test_pipe Init.Pipe.stderr);
  ]

let () = Alcotest.run "sdk" [
    "init", test;
  ]
