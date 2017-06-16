open Lwt.Infix
open Capnp_rpc_lwt

let connect ~switch path =
  Logs.info (fun f -> f "Connecting to %S" path);
  let socket = Unix.(socket PF_UNIX SOCK_STREAM 0) in
  begin
    try Unix.connect socket (Unix.ADDR_UNIX path)
    with Unix.Unix_error(Unix.ECONNREFUSED, "connect", "") ->
    Logs.err (fun f -> f "Failed to connect to %S" path);
    exit 1
  end;
  let endpoint = Endpoint.of_socket ~switch socket in
  let conn = CapTP.of_endpoint ~switch endpoint in
  CapTP.bootstrap conn

let rm_socket path =
  match Unix.lstat path with
  | stat when stat.Unix.st_kind = Unix.S_SOCK -> Unix.unlink path
  | _ -> failwith (Fmt.strf "%S exists and is not a socket" path)
  | exception Unix.Unix_error(Unix.ENOENT, "lstat", _) -> ()

let listen ~switch ~offer path =
  let socket = Unix.(socket PF_UNIX SOCK_STREAM 0) in
  Lwt_switch.add_hook (Some switch) (fun () -> Unix.close socket; Lwt.return_unit);
  rm_socket path;
  Unix.bind socket (Unix.ADDR_UNIX path);
  Unix.listen socket 5;
  let socket = Lwt_unix.of_unix_file_descr socket in
  Logs.info (fun f -> f "Waiting for connections on %S" path);
  let rec loop () =
    Lwt_unix.accept socket >>= fun (c, _) ->
    Logs.info (fun f -> f "Got connection on %S" path);
    Lwt_switch.with_switch @@ fun switch ->     (* todo: with_child_switch *)
    let endpoint = Endpoint.of_socket ~switch (Lwt_unix.unix_file_descr c) in
    ignore (CapTP.of_endpoint ~switch ~offer endpoint);
    loop () in
  loop ()

module Actor = struct
  type t = Fmt.style * string
  let pp f (style, name) = Fmt.(styled style (const string name)) f ()
  let tag = Logs.Tag.def "actor" pp
end

let pp_qid f = function
  | None -> ()
  | Some x ->
    let s = Uint32.to_string x in
    Fmt.(styled `Magenta (fun f x -> Fmt.pf f " (qid=%s)" x)) f s

let reporter =
  let report src level ~over k msgf =
    let src = Logs.Src.name src in
    msgf @@ fun ?header ?(tags=Logs.Tag.empty) fmt ->
    let actor =
      match Logs.Tag.find Actor.tag tags with
      | Some x -> x
      | None -> `Black, "------"
    in
    let qid = Logs.Tag.find Capnp_rpc.Debug.qid_tag tags in
    let print _ =
      Fmt.(pf stderr) "%a@." pp_qid qid;
      over ();
      k ()
    in
    Fmt.kpf print Fmt.stderr ("%a %a %a: @[" ^^ fmt ^^ "@]")
      Fmt.(styled `Magenta string) (Printf.sprintf "%11s" src)
      Logs_fmt.pp_header (level, header)
      Actor.pp actor
  in
  { Logs.report = report }

let init_logging () =
  Fmt_tty.setup_std_outputs ();
  Logs.set_reporter reporter;
  Logs.set_level (Some Logs.Info)
