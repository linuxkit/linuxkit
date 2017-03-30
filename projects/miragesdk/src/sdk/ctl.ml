open Lwt.Infix
open Astring

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

module Query = struct

  (* FIXME: this should probably be replaced by protobuf *)

  [%%cenum
    type operation =
      | Write
      | Read
      | Delete
    [@@uint8_t]
  ]

  type t = {
    version  : int32;
    id       : int32;
    operation: operation;
    path     : string;
    payload  : string;
  }

  [%%cstruct type msg = {
      version   : uint32_t; (* protocol version *)
      id        : uint32_t; (* session identifier *)
      operation : uint8_t;  (* = type operation *)
      path      : uint16_t;
      payload   : uint32_t;
    } [@@little_endian]
  ]

  (* to avoid warning 32 *)
  let _ = hexdump_msg
  let _ = string_to_operation

  let pp ppf t =
    Fmt.pf ppf "%ld:%s:%S:%S"
      t.id (operation_to_string t.operation) t.path t.payload

  (* FIXME: allocate less ... *)

  let of_cstruct buf =
    let open Rresult.R in
    Log.debug (fun l -> l "Query.of_cstruct %S" @@ Cstruct.to_string buf);
    let version = get_msg_version buf in
    let id = get_msg_id buf in
    (match int_to_operation (get_msg_operation buf) with
     | None   -> Error (`Msg "invalid operation")
     | Some o -> Ok o)
    >>= fun operation ->
    let path_len = get_msg_path buf in
    let payload_len = get_msg_payload buf in
    let path =
      Cstruct.sub buf sizeof_msg path_len
      |> Cstruct.to_string
    in
    let payload =
      Cstruct.sub buf (sizeof_msg + path_len) (Int32.to_int payload_len)
      |> Cstruct.to_string
    in
    if String.Ascii.is_valid path then Ok { version; id; operation; path; payload }
    else Error (`Msg "invalid path")

  let to_cstruct msg =
    Log.debug (fun l -> l "Query.to_cstruct %a" pp msg);
    let operation = operation_to_int msg.operation in
    let path = String.length msg.path in
    let payload = String.length msg.payload in
    let len = sizeof_msg + path + payload in
    let buf = Cstruct.create len in
    set_msg_version buf msg.version;
    set_msg_id buf msg.id;
    set_msg_operation buf operation;
    set_msg_path buf path;
    set_msg_payload buf (Int32.of_int payload);
    Cstruct.blit_from_bytes msg.path 0 buf sizeof_msg path;
    Cstruct.blit_from_bytes msg.payload 0 buf (sizeof_msg+path) payload;
    buf

  let read fd =
    IO.read_n fd 4 >>= fun buf ->
    Log.debug (fun l -> l "Message.read len=%S" buf);
    let len =
      Cstruct.LE.get_uint32 (Cstruct.of_string buf) 0
      |> Int32.to_int
    in
    IO.read_n fd len >|= fun buf ->
    of_cstruct (Cstruct.of_string buf)

  let write fd msg =
    let buf = to_cstruct msg |> Cstruct.to_string in
    let len =
      let len = Cstruct.create 4 in
      Cstruct.LE.set_uint32 len 0 (Int32.of_int @@ String.length buf);
      Cstruct.to_string len
    in
    IO.write fd len >>= fun () ->
    IO.write fd buf

end

module Reply = struct

  (* FIXME: this should probably be replaced by protobuf *)

  [%%cenum
    type status =
      | Ok
      | Error
    [@@uint8_t]
  ]

  type t = {
    id     : int32;
    status : status;
    payload: string;
  }

  [%%cstruct type msg = {
      id     : uint32_t; (* session identifier *)
      status : uint8_t;  (* = type operation *)
      payload: uint32_t;
    } [@@little_endian]
  ]

  (* to avoid warning 32 *)
  let _ = hexdump_msg
  let _ = string_to_status

  let pp ppf t =
    Fmt.pf ppf "%ld:%s:%S" t.id (status_to_string t.status) t.payload

  (* FIXME: allocate less ... *)

  let of_cstruct buf =
    let open Rresult.R in
    Log.debug (fun l -> l "Message.of_cstruct %S" @@ Cstruct.to_string buf);
    let id = get_msg_id buf in
    (match int_to_status (get_msg_status buf) with
     | None   -> Error (`Msg "invalid operation")
     | Some o -> Ok o)
    >>= fun status ->
    let payload_len = Int32.to_int (get_msg_payload buf) in
    let payload =
      Cstruct.sub buf sizeof_msg payload_len
      |> Cstruct.to_string
    in
    Ok { id; status; payload }

  let to_cstruct msg =
    Log.debug (fun l -> l "Message.to_cstruct %a" pp msg);
    let status = status_to_int msg.status in
    let payload = String.length msg.payload in
    let len = sizeof_msg + payload in
    let buf = Cstruct.create len in
    set_msg_id buf msg.id;
    set_msg_status buf status;
    set_msg_payload buf (Int32.of_int payload);
    Cstruct.blit_from_bytes msg.payload 0 buf sizeof_msg payload;
    buf

  let read fd =
    IO.read_n fd 4 >>= fun buf ->
    Log.debug (fun l -> l "Message.read len=%S" buf);
    let len =
      Cstruct.LE.get_uint32 (Cstruct.of_string buf) 0
      |> Int32.to_int
    in
    IO.read_n fd len >|= fun buf ->
    of_cstruct (Cstruct.of_string buf)

  let write fd msg =
    let buf = to_cstruct msg |> Cstruct.to_string in
    let len =
      let len = Cstruct.create 4 in
      Cstruct.LE.set_uint32 len 0 (Int32.of_int @@ String.length buf);
      Cstruct.to_string len
    in
    IO.write fd len >>= fun () ->
    IO.write fd buf

end

let err_not_found = "err-not-found"

module Client = struct

  let new_id =
    let n = ref 0l in
    fun () -> n := Int32.succ !n; !n

  let version = 0l

  module K = struct
    type t = int32
    let equal = Int32.equal
    let hash = Hashtbl.hash
  end
  module Cache = Hashtbl.Make(K)

  type t = {
    fd     : Lwt_unix.file_descr;
    replies: Reply.t Cache.t;
  }

  let v fd = { fd; replies = Cache.create 12 }

  let call t query =
    let id = query.Query.id in
    Query.write t.fd query >>= fun () ->
    let rec loop () =
      try
        let r = Cache.find t.replies id in
        Cache.remove t.replies id;
        Lwt.return r
      with Not_found ->
        Reply.read t.fd >>= function
        | Error (`Msg e) ->
          Log.err (fun l -> l "Got %s while waiting for a reply to %ld" e id);
          loop ()
        | Ok r ->
          if r.id = id then Lwt.return r
          else (
            (* FIXME: maybe we want to check if id is not already
               allocated *)
            Cache.add t.replies r.id r;
            loop ()
          )
    in
    loop () >|= fun r ->
    assert (r.Reply.id = id);
    match r.Reply.status with
    | Ok    -> Ok r.Reply.payload
    | Error -> Error (`Msg r.Reply.payload)

  let query operation path payload =
    let id = new_id () in
    { Query.version; id; operation; path; payload }

  let read t path =
    call t (query Read path "") >|= function
    | Ok x           -> Ok (Some x)
    | Error (`Msg e) ->
      if e = err_not_found then Ok None
      else Error (`Msg e)

  let write t path v =
    call t (query Write path v) >|= function
    | Ok ""        -> Ok ()
    | Ok _         -> Error (`Msg "invalid return")
    | Error _ as e -> e

  let delete t path =
    call t (query Delete path "") >|= function
    | Ok ""        -> Ok ()
    | Ok _         -> Error (`Msg "invalid return")
    | Error _ as e -> e

end

module Server = struct

  let ok q payload =
    { Reply.id = q.Query.id; status = Reply.Ok; payload }

  let error q payload =
    { Reply.id = q.Query.id; status = Reply.Error; payload }

  let with_key q f =
    match KV.Key.of_string q.Query.path with
    | Ok x           -> f x
    | Error (`Msg e) ->
      Fmt.kstrf (fun msg -> Lwt.return (error q msg)) "invalid key: %s" e

  let infof fmt =
    Fmt.kstrf (fun msg () ->
        let date = Int64.of_float (Unix.gettimeofday ()) in
        Irmin.Info.v ~date ~author:"calf" msg
      ) fmt

  let dispatch db q =
    with_key q (fun key ->
        match q.Query.operation with
        | Write ->
          let info = infof "Updating %a" KV.Key.pp key in
          KV.set db ~info key q.payload >|= fun () ->
          ok q ""
        | Delete ->
          let info = infof "Removing %a" KV.Key.pp key in
          KV.remove db ~info key >|= fun () ->
          ok q ""
        | Read ->
          KV.find db key >|= function
          | None   -> error q err_not_found
          | Some v -> ok q v
      )


  let int_of_fd (t:Lwt_unix.file_descr) =
    (Obj.magic (Lwt_unix.unix_file_descr t): int)

  let listen ~routes db fd =
    Lwt_unix.blocking fd >>= fun blocking ->
    Log.debug (fun l ->
        l "Serving the control state over fd:%d (blocking=%b)"
          (int_of_fd fd) blocking
      );
    let queries = Queue.create () in
    let cond = Lwt_condition.create () in
    let rec listen () =
      Query.read fd >>= function
      | Error (`Msg e) ->
        Log.err (fun l -> l "received invalid message: %s" e);
        listen ()
      | Ok q ->
        Queue.add q queries;
        Lwt_condition.signal cond ();
        listen ()
    in
    let rec process () =
      Lwt_condition.wait cond >>= fun () ->
      let q = Queue.pop queries in
      let path = q.Query.path in
      (if List.mem path routes then (
          dispatch db q >>= fun r ->
          Reply.write fd r
        ) else (
         let err = Fmt.strf "%s is not an allowed path" path in
         Log.err (fun l -> l "%ld: %s" q.Query.id path);
         Reply.write fd (error q err)
       )) >>= fun () ->
      process ()
    in
    Lwt.pick [
      listen ();
      process ();
    ]

end
