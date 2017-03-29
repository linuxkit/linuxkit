open Lwt.Infix

let src = Logs.Src.create "init" ~doc:"Init steps"
module Log = (val Logs.src_log src : Logs.LOG)

(* FIXME: to avoid linking with gmp *)
module No_IO = struct
  type ic = unit
  type oc = unit
  type ctx = unit
  let with_connection ?ctx:_ _uri ?init:_ _f = Lwt.fail_with "not allowed"
  let read_all _ic = Lwt.fail_with "not allowed"
  let read_exactly _ic _n = Lwt.fail_with "not allowed"
  let write _oc _buf = Lwt.fail_with "not allowed"
  let flush _oc = Lwt.fail_with "not allowed"
  let ctx () = Lwt.return_none
end

(* FIXME: we don't use Irmin_unix.Git.FS.KV to avoid linking with gmp *)
module Store = Irmin_git.FS.KV(No_IO)(Inflator)(Io_fs)
module KV = Store(Irmin.Contents.String)

let v path =
  let config = Irmin_git.config path in
  KV.Repo.v config >>= fun repo ->
  KV.of_branch repo "calf"

let () =
  Irmin.Private.Watch.set_listen_dir_hook Irmin_watcher.hook

module Message = struct

  [%%cenum
    type operation =
      | Write
      | Read
      | Delete
    [@@uint8_t]
  ]

  type t = {
    operation: operation;
    path     : string;
    payload  : string option;
  }

  [%%cstruct type message = {
      operation : uint8_t; (* = type operation *)
      path      : uint16_t;
      payload   : uint16_t;
    } [@@little_endian]
  ]

  (* to avoid warning 32 *)
  let _ = hexdump_message
  let _ = operation_to_string
  let _ = string_to_operation

  let read_message fd =
    IO.read_n fd 4 >>= fun buf ->
    let len =
      Cstruct.LE.get_uint32 (Cstruct.of_string buf) 0
      |> Int32.to_int
    in
    IO.read_n fd len >>= fun buf ->
    let buf = Cstruct.of_string buf in
    let operation = match int_to_operation (get_message_operation buf) with
      | None   -> failwith "invalid operation"
      | Some o -> o
    in
    let path_len = get_message_path buf in
    let payload_len = get_message_payload buf in
    IO.read_n fd path_len >>= fun path ->
    (match payload_len with
     | 0 -> Lwt.return None
     | n -> IO.read_n fd n >|= fun x -> Some x)
    >|= fun payload ->
    { operation; path; payload }

  let write_message fd msg =
    let operation = operation_to_int msg.operation in
    let path = String.length msg.path in
    let payload = match msg.payload with
      | None   -> 0
      | Some x -> String.length x
    in
    let len = sizeof_message + path + payload in
    let buf = Cstruct.create len in
    set_message_operation buf operation;
    set_message_path buf path;
    set_message_payload buf path;
    Cstruct.blit_from_bytes msg.path 0 buf sizeof_message path;
    let () = match msg.payload with
      | None   -> ()
      | Some x -> Cstruct.blit_from_bytes x 0 buf (sizeof_message+path) payload
    in
    IO.really_write fd (Cstruct.to_string buf) 0 len

end

module Dispatch = struct

  open Message

  let with_key msg f =
    match KV.Key.of_string msg.path with
    | Ok x           -> f x
    | Error (`Msg e) -> Fmt.kstrf Lwt.fail_with "invalid key: %s" e

  let infof fmt =
    Fmt.kstrf (fun msg () ->
        let date = Int64.of_float (Unix.gettimeofday ()) in
        Irmin.Info.v ~date ~author:"calf" msg
      ) fmt

  let dispatch db msg =
    with_key msg (fun key ->
        match msg.operation with
        | Write ->
          let info = infof "Updating %a" KV.Key.pp key in
          (match msg.payload with
           | None   -> Fmt.kstrf Lwt.fail_with "dispatch: missing payload"
           | Some v -> KV.set db ~info key v)
        | _ -> failwith "TODO"
      )

  let serve fd db ~routes =
    let msgs = Queue.create () in
    let cond = Lwt_condition.create () in
    let rec listen () =
      read_message fd >>= fun msg ->
      Queue.add msg msgs;
      Lwt_condition.signal cond ();
      listen ()
    in
    let rec process () =
      Lwt_condition.wait cond >>= fun () ->
      let msg = Queue.pop msgs in
      (if List.mem msg.path routes then dispatch db msg
       else (
         Log.err (fun l -> l "%s is not an allowed path" msg.path);
         Lwt.return_unit;
       )) >>= fun () ->
      process ()
    in
    Lwt.pick [
      listen ();
      process ();
    ]

end

let int_of_fd (t:Lwt_unix.file_descr) =
  (Obj.magic (Lwt_unix.unix_file_descr t): int)

let serve ~routes db fd =
  Lwt_unix.blocking fd >>= fun blocking ->
  Log.debug (fun l ->
      l "Serving the control state over fd:%d (blocking=%b)"
        (int_of_fd fd) blocking
    );
  Dispatch.serve fd db ~routes
