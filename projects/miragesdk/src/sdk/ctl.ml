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
  Irmin.Private.Watch.set_listen_dir_hook
    (fun _ _ _ -> Lwt.return (fun () -> Lwt.return_unit))
    (* FIXME: inotify need some unknown massaging. *)
    (* Irmin_watcher.hook *)

module C = Mirage_channel_lwt.Make(IO)

module P = Proto.Make(Capnp.BytesMessage)

exception Undefined_field of int

module Endpoint = struct

  let compression = `None

  type t = {
    output : IO.t;
    input  : C.t;  (* reads are buffered *)
    decoder: Capnp.Codecs.FramedStream.t;
  }

  type error = [
    | `IO of IO.write_error
    | `Channel of C.error
    | `Msg of string
    | `Undefined_field of int
  ]

  let pp_error ppf (e:error) = match e with
    | `IO e              -> Fmt.pf ppf "IO: %a" IO.pp_write_error e
    | `Channel e         -> Fmt.pf ppf "channel: %a" C.pp_error e
    | `Msg e             -> Fmt.string ppf e
    | `Undefined_field i -> Fmt.pf ppf "undefined field %d" i

  let err_io e = Error (`IO e)
  let err_channel e = Error (`Channel e)
  let err_msg fmt = Fmt.kstrf (fun s -> Error (`Msg s)) fmt
  let err_frame = err_msg "Unsupported Cap'n'Proto frame received"
  let err_undefined_field i = Error (`Undefined_field i)

  let v fd =
    let output = fd in
    let input = C.create fd in
    let decoder = Capnp.Codecs.FramedStream.empty compression in
    { output; input; decoder }

  let send t msg =
    let buf = Capnp.Codecs.serialize ~compression msg in
    (* FIXME: avoid copying *)
    IO.write t.output (Cstruct.of_string buf) >|= function
    | Error e -> err_io e
    | Ok ()   -> Ok ()

  let rec recv t =
    match Capnp.Codecs.FramedStream.get_next_frame t.decoder with
    | Ok msg                                      -> Lwt.return (Ok (`Data msg))
    | Error Capnp.Codecs.FramingError.Unsupported ->  Lwt.return err_frame
    | Error Capnp.Codecs.FramingError.Incomplete  ->
      Log.info (fun f -> f "Endpoint.recv: incomplete; waiting for more data");
      C.read_some ~len:4096 t.input >>= function
      | Ok `Eof         -> Lwt.return (Ok `Eof)
      | Error e         -> Lwt.return (err_channel e)
      | Ok (`Data data) ->
        (* FIXME: avoid copying *)
        let data = Cstruct.to_string data in
        Log.info (fun f -> f "Got %S" data);
        Capnp.Codecs.FramedStream.add_fragment t.decoder data;
        recv t

end

module Request = struct

  type action =
    | Write of string
    | Read
    | Delete

  let pp_action ppf = function
    | Write s -> Fmt.pf ppf "write[%S]" s
    | Read    -> Fmt.pf ppf "read"
    | Delete  -> Fmt.pf ppf "delete"

  type t = {
    id    : int32 Lazy.t;
    path  : string list Lazy.t;
    action: action Lazy.t;
  }

  let id t = Lazy.force t.id
  let path t = Lazy.force t.path
  let action t = Lazy.force t.action

  let pp_path = Fmt.(list ~sep:(unit "/") string)

  let pp ppf t =
    let id = id t and path = path t and action = action t in
    match action with
    | exception Undefined_field i -> Fmt.pf ppf "<undefined-field: %d>" i
    | action -> Fmt.pf ppf "%ld:%a:%a" id pp_path path pp_action action

  let equal x y =
    id x = id y && path x = path y && match action x = action y with
    | exception Undefined_field _ -> false
    | b -> b

  let v ~id ~path action =
    { id = lazy id; action = lazy action; path = lazy path }

  let read e: (t, Endpoint.error) result Lwt.t =
    Endpoint.recv e >|= function
    | Error e      -> Error e
    | Ok `Eof      -> Error (`IO `Closed)
    | Ok (`Data x) ->
      let open P.Reader in
      let msg = Request.of_message x in
      let id = lazy (Request.id_get msg) in
      let path = lazy (Request.path_get_list msg) in
      let action = lazy (match Request.get msg with
          | Request.Write x     -> Write x
          | Request.Read        -> Read
          | Request.Delete      -> Delete
          | Request.Undefined i -> raise (Undefined_field i)
        ) in
      Ok { id; path; action }

  let write e t =
    let open P.Builder in
    match action t with
    | exception Undefined_field i -> Lwt.return (Endpoint.err_undefined_field i)
    | action ->
      let msg =
        let b = Request.init_root () in
        Request.id_set b (id t);
        ignore (Request.path_set_list b (path t));
        (match action with
         | Write x -> Request.write_set b x
         | Read    -> Request.read_set b
         | Delete  -> Request.delete_set b);
        b
      in
      Endpoint.send e (Request.to_message msg)

end

module Response = struct

  type status = (string, string) result

  let pp_status ppf = function
    | Ok ok   -> Fmt.pf ppf "ok:%S" ok
    | Error e -> Fmt.pf ppf "error:%S" e

  type t = {
    id    : int32 Lazy.t;
    status: status Lazy.t;
  }

  let v ~id status = { id = lazy id; status = lazy status }
  let id t = Lazy.force t.id
  let status t = Lazy.force t.status

  let pp ppf t = match status t with
    | exception Undefined_field i -> Fmt.pf ppf "<undefined-field %d>" i
    | s -> Fmt.pf ppf "%ld:%a" (id t) pp_status s

  let equal x y =
    id x = id y && match status x = status y with
    | exception Undefined_field _ -> false
    | b -> b

  let read e: (t, Endpoint.error) result Lwt.t =
    Endpoint.recv e >|= function
    | Error e      -> Error e
    | Ok `Eof      -> Error (`IO `Closed)
    | Ok (`Data x) ->
      let open P.Reader in
      let msg = Response.of_message x in
      let id = lazy (Response.id_get msg) in
      let status = lazy (match Response.get msg with
          | Response.Ok x        -> Ok x
          | Response.Error x     -> Error x
          | Response.Undefined i -> raise (Undefined_field i)
        ) in
      Ok { id; status }

  let write e t =
    let open P.Builder in
    match status t with
    | exception Undefined_field i -> Lwt.return (Endpoint.err_undefined_field i)
    | s ->
      let msg =
        let b = Response.init_root () in
        Response.id_set b (id t);
        (match s with
         | Error s -> Response.error_set b s
         | Ok s    -> Response.ok_set b s);
        b
      in
      Endpoint.send e (Response.to_message msg)

end

let err_not_found = "err-not-found"

module Client = struct

  let new_id =
    let n = ref 0l in
    fun () -> n := Int32.succ !n; !n

  type error = [`Msg of string]
  let pp_error ppf (`Msg s) = Fmt.string ppf s

  module K = struct
    type t = int32
    let equal = Int32.equal
    let hash = Hashtbl.hash
  end
  module Cache = Hashtbl.Make(K)

  type t = {
    e      : Endpoint.t;
    replies: Response.t Cache.t;
  }

  let v fd = { e = Endpoint.v fd; replies = Cache.create 12 }
  let err e = Fmt.kstrf (fun e -> Error (`Msg e)) "%a" Endpoint.pp_error e

  let call t r =
    let id = Request.id r in
    Request.write t.e r >>= function
    | Error e -> Lwt.return (err e)
    | Ok ()   ->
      let rec loop () =
        try
          let r = Cache.find t.replies id in
          Cache.remove t.replies id;
          Lwt.return r
        with Not_found ->
          Response.read t.e >>= function
          | Error e ->
            Log.err (fun l -> l "Got %a while waiting for a reply to %ld"
                        Endpoint.pp_error e id);
            loop ()
          | Ok r ->
            let rid = Response.id r in
            if rid = id then Lwt.return r
            else (
              (* FIXME: maybe we want to check if id is not already
                 allocated *)
              Cache.add t.replies rid r;
              loop ()
            )
      in
      loop () >|= fun r ->
      assert (Response.id r = id);
      match Response.status r with
      | Ok s    -> Ok s
      | Error s -> Error (`Msg s)

  let request path action =
    let id = new_id () in
    Request.v ~id ~path action

  let read t path =
    call t (request path Read) >|= function
    | Ok x    -> Ok (Some x)
    | Error e ->
      if e = `Msg err_not_found then Ok None
      else Error e

  let write t path v =
    call t (request path @@ Write v) >|= function
    | Ok ""        -> Ok ()
    | Ok _         -> Error (`Msg "invalid return")
    | Error _ as e -> e

  let delete t path =
    call t (request path Delete) >|= function
    | Ok ""        -> Ok ()
    | Ok _         -> Error (`Msg "invalid return")
    | Error _ as e -> e

end

module Server = struct

  type op = [ `Read | `Write | `Delete ]

  let ok q s = Response.v ~id:(Request.id q) (Ok s)
  let error q s = Response.v ~id:(Request.id q) (Error s)
  let with_key q f = f (Request.path q)

  let infof fmt =
    Fmt.kstrf (fun msg () ->
        let date = Int64.of_float (Unix.gettimeofday ()) in
        Irmin.Info.v ~date ~author:"calf" msg
      ) fmt

  let not_allowed q =
    let path = Request.path q in
    let err = Fmt.strf "%a is not an allowed path" Request.pp_path path in
    Log.err (fun l -> l "%ld: %a" (Request.id q) Request.pp_path path);
    error q err

  let dispatch db op q =
    with_key q (fun key ->
        let can x = List.mem x op in
        match Request.action q with
        | exception Undefined_field i ->
          Fmt.kstrf (fun e -> Lwt.return (error q e)) "undefined field %i" i
        | Write s when can `Write ->
          let info = infof "Updating %a" KV.Key.pp key in
          KV.set db ~info key s >|= fun () ->
          ok q ""
        | Delete when can `Delete ->
          let info = infof "Removing %a" KV.Key.pp key in
          KV.remove db ~info key >|= fun () ->
          ok q ""
        | Read when can `Read ->
          (KV.find db key >|= function
          | None   -> error q err_not_found
          | Some v -> ok q v)
        | _ -> Lwt.return (not_allowed q)
      )

  let listen ~routes db fd =
    Log.debug (fun l -> l "Serving the control state over %a" IO.pp fd);
    let queries = Queue.create () in
    let cond = Lwt_condition.create () in
    let e = Endpoint.v fd in
    let rec listen () =
      Request.read e >>= function
      | Error (`Channel _ | `IO _ as e) ->
        Log.err (fun l -> l "fatal error: %a" Endpoint.pp_error e);
        Lwt.return_unit
      | Error (`Msg _ | `Undefined_field _ as e) ->
        Log.err (fun l -> l "transient error: %a" Endpoint.pp_error e);
        listen ()
      | Ok q ->
        Queue.add q queries;
        Lwt_condition.signal cond ();
        listen ()
    in
    let rec process () =
      Lwt_condition.wait cond >>= fun () ->
      let q = Queue.pop queries in
      let path = Request.path q in
      (if List.mem_assoc path routes then (
          let op = List.assoc path routes in
          dispatch db op q >>= fun r ->
          Response.write e r
        ) else (
         Response.write e (not_allowed q)
       )) >>= function
      | Ok ()   -> process ()
      | Error e ->
        Log.err (fun l -> l "%a" Endpoint.pp_error e);
        process ()
    in
    Lwt.pick [
      listen ();
      process ();
    ]

end
