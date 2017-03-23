open Lwt.Infix

let src = Logs.Src.create "dhcp-client" ~doc:"DHCP client"
module Log = (val Logs.src_log src : Logs.LOG)

let failf fmt = Fmt.kstrf Lwt.fail_with fmt

type fd = {
  name: string;
  fd  : Lwt_unix.file_descr;
}

let stdout = { name = "stdout"; fd = Lwt_unix.stdout }
let stderr = { name = "stderr"; fd = Lwt_unix.stderr }
let stdin  = { name = "stdin" ; fd = Lwt_unix.stdin  }

let int_of_fd (fd:Lwt_unix.file_descr) =
  (Obj.magic (Lwt_unix.unix_file_descr fd): int)

let pp_fd ppf fd = Fmt.pf ppf "%s:%d" fd.name (int_of_fd fd.fd)

let close fd =
  Log.debug (fun l -> l "close %a" pp_fd fd);
  Lwt_unix.close fd.fd

let dev_null =
  Lwt_unix.of_unix_file_descr ~blocking:false
    (Unix.openfile "/dev/null" [Unix.O_RDWR] 0)

let close_and_dup fd =
  Log.debug (fun l -> l "close-and-dup %a" pp_fd fd);
  Lwt_unix.close fd.fd >>= fun () ->
  Lwt_unix.dup2 dev_null fd.fd;
  Lwt_unix.close dev_null

let dup2 ~src ~dst =
  Log.debug (fun l -> l "dup2 %a => %a" pp_fd src pp_fd dst);
  Lwt_unix.dup2 src.fd dst.fd;
  close src

let proxy_rawlink ~rawlink ~fd =
  Log.debug (fun l -> l "proxy-netif eth0 <=> %a" pp_fd fd);
  let rec listen_rawlink () =
    Lwt_rawlink.read_packet rawlink >>= fun buf ->
    Log.debug (fun l -> l "PROXY-NETIF: => %a" Cstruct.hexdump_pp buf);
    Log.debug (fun l -> l "PROXY-NETIF: => %S" (Cstruct.to_string buf));
    let rec write buf =
      Lwt_cstruct.write fd.fd buf >>= function
      | 0 -> Lwt.return_unit
      | n -> write (Cstruct.shift buf n)
    in
    write buf >>= fun () ->
    listen_rawlink ()
  in
  let listen_socket () =
    let len = 16 * 1024 in
    let buf = Cstruct.create len in
    let rec loop () =
      Lwt_cstruct.read fd.fd buf >>= fun len ->
      let buf = Cstruct.sub buf 0 len in
      Log.debug (fun l -> l "PROXY-NETIF: <= %a" Cstruct.hexdump_pp buf);
      Lwt_rawlink.send_packet rawlink buf >>= fun () ->
      loop ()
    in
    loop ()
  in
  Lwt.pick [
    listen_rawlink ();
    listen_socket ();
  ]

let rec really_write dst buf off len =
  match len with
  | 0   -> Lwt.return_unit
  | len ->
    Lwt_unix.write dst.fd buf off len >>= fun n ->
    really_write dst buf (off+n) (len-n)

let forward ~src ~dst =
  Log.debug (fun l -> l "forward %a => %a" pp_fd src pp_fd dst);
  let len = 16 * 1024 in
  let buf = Bytes.create len in
  let rec loop () =
    Lwt_unix.read src.fd buf 0 len >>= fun len ->
    if len = 0 then
      (* FIXME: why this ever happen *)
      Fmt.kstrf Lwt.fail_with "FORWARD[%a => %a]: EOF" pp_fd src pp_fd dst
    else (
      Log.debug (fun l ->
          l "FORWARD[%a => %a]: %S (%d)"
            pp_fd src pp_fd dst (Bytes.sub buf 0 len) len);
      really_write dst buf 0 len >>= fun () ->
      loop ()
    )
  in
  loop ()

let proxy x y =
  Lwt.pick [
    forward ~src:x ~dst:y;
    forward ~src:y ~dst:x;
  ]

(* Prepare the fd space before we fork to run the calf *)

let socketpair name =
  let priv, calf = Lwt_unix.(socketpair PF_UNIX SOCK_STREAM 0) in
  Lwt_unix.clear_close_on_exec priv;
  Lwt_unix.clear_close_on_exec calf;
  { name = name; fd = priv }, { name = name ^ "-calf"; fd = calf }

let pipe name =
  let priv, calf = Lwt_unix.pipe () in
  Lwt_unix.clear_close_on_exec priv;
  Lwt_unix.clear_close_on_exec calf;
  { name = name; fd = priv }, { name = name ^ "-calf"; fd = calf }

(* logs pipe *)
let logs_out = pipe "logs-out"
let logs_err = pipe "logs-err"

(* store pipe *)
let store = socketpair "store"

(* network pipe *)
let net = socketpair "net"

(* metrics pipe *)
(* let metrics = make "metrics" *)

let child cmd =
  close_and_dup stdin  >>= fun () ->

  (* close parent fds *)
  close (fst logs_out) >>= fun () ->
  close (fst logs_err) >>= fun () ->
  close (fst store)    >>= fun () ->
  close (fst net)      >>= fun () ->
  (*
    close (fst metrics) >>= fun () ->
  *)

  let cmds = String.concat " " cmd in
  Log.info (fun l -> l "Executing %s" cmds);
  Log.debug (fun l ->
      l "net-fd=%a store-fd=%a" pp_fd (snd net) pp_fd (snd store));

  dup2 ~src:(snd logs_out) ~dst:stdout >>= fun () ->
  dup2 ~src:(snd logs_err) ~dst:stderr >>= fun () ->

  (* exec the calf *)
  Unix.execve (List.hd cmd) (Array.of_list cmd) [||]

module Store = struct

  (* FIXME: to avoid linking with gmp *)
  module IO = struct
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
  module Store = Irmin_git.FS.KV(IO)(Inflator)(Io_fs)
  module KV = Store(Irmin.Contents.String)

  let client () =
    let config = Irmin_git.config "/data" in
    KV.Repo.v config >>= fun repo ->
    KV.of_branch repo "calf"

  let set_listen_dir_hook () =
    Irmin.Private.Watch.set_listen_dir_hook Irmin_watcher.hook

  module HTTP = struct

    module Wm = struct
      module Rd = Webmachine.Rd
      include Webmachine.Make(Cohttp_lwt_unix.Server.IO)
    end

    let with_key rd f =
      match KV.Key.of_string rd.Wm.Rd.dispatch_path with
      | Ok x    -> f x
      | Error _ -> Wm.respond 404 rd

    let infof fmt =
      Fmt.kstrf (fun msg () ->
          let date = Int64.of_float (Unix.gettimeofday ()) in
          Irmin.Info.v ~date ~author:"calf" msg
        ) fmt

    let ok = "{\"status\": \"ok\"}"

    class item db = object(self)

      inherit [Cohttp_lwt_body.t] Wm.resource

      method private of_string rd =
        Cohttp_lwt_body.to_string rd.Wm.Rd.req_body >>= fun value ->
        with_key rd (fun key ->
            let info = infof "Updating %a" KV.Key.pp key in
            KV.set db ~info key value >>= fun () ->
            let resp_body = `String ok in
            let rd = { rd with Wm.Rd.resp_body } in
            Wm.continue true rd
          )

      method private to_string rd =
        with_key rd (fun key ->
            KV.find db key >>= function
            | Some value -> Wm.continue (`String value) rd
            | None       -> assert false
          )

      method resource_exists rd =
        with_key rd (fun key ->
            KV.mem db key >>= fun mem ->
            Wm.continue mem rd
          )

      method allowed_methods rd =
        Wm.continue [`GET; `HEAD; `PUT; `DELETE] rd

      method content_types_provided rd =
        Wm.continue [
          "plain", self#to_string
        ] rd

      method content_types_accepted rd =
        Wm.continue [
          "plain", self#of_string
        ] rd

      method delete_resource rd =
        with_key rd (fun key ->
            let info = infof "Deleting %a" KV.Key.pp key in
            KV.remove db ~info key >>= fun () ->
            let resp_body = `String ok in
            Wm.continue true { rd with Wm.Rd.resp_body }
          )
    end

    let v db =
      let routes = [
        ("/ip"          , fun () -> new item db);
        ("/domain"      , fun () -> new item db);
        ("/search"      , fun () -> new item db);
        ("/mtu"         , fun () -> new item db);
        ("/nameserver/*", fun () -> new item db);
      ] in
      let callback (_ch, _conn) request body =
        let open Cohttp in
        (Wm.dispatch' routes ~body ~request >|= function
          | None        -> (`Not_found, Header.init (), `String "Not found", [])
          | Some result -> result)
        >>= fun (status, headers, body, path) ->
        Log.info (fun l ->
            l "%d - %s %s"
              (Code.code_of_status status)
              (Code.string_of_method (Request.meth request))
              (Uri.path (Request.uri request)));
        Log.debug (fun l -> l "path=%a" Fmt.(Dump.list string) path);
        (* Finally, send the response to the client *)
        Cohttp_lwt_unix.Server.respond ~flush:true ~headers ~body ~status ()
      in
      (* create the server and handle requests with the function defined above *)
      let conn_closed (_, conn) =
        Log.info (fun l ->
            l "connection %s closed\n%!" (Cohttp.Connection.to_string conn))
      in
      Cohttp_lwt_unix.Server.make ~callback ~conn_closed ()
    end

  let serve () =
    client () >>= fun db ->
    let http = HTTP.v db in
    let fd = fst store in
    let on_exn e = Log.err (fun l -> l "XXX %a" Fmt.exn e) in
    Log.info (fun l -> l "serving KV store on %a" pp_fd fd);
    Cohttp_lwt_unix.Server.create ~on_exn ~mode:(`Fd fd.fd) http

end

module Handlers = struct

  (* System handlers *)

  let contents_of_diff = function
    | `Added (_, `Contents (v, _))
    | `Updated (_, (_, `Contents (v, _))) -> Some v
    | _ -> None

  let ip t =
    Store.KV.watch_key t ["ip"] (fun diff ->
        match contents_of_diff diff with
        | Some ip ->
          Log.info (fun l -> l "SET IP to %s" ip);
          Lwt.return ()
        | _ ->
          Lwt.return ()
      )

  let handlers = [
    ip;
  ]

  let install () =
    Store.client () >>= fun db ->
    Lwt_list.map_p (fun f -> f db) handlers >>= fun _ ->
    let t, _ = Lwt.task () in
    t

end

external bpf_filter: unit -> string = "bpf_filter"

let rawlink ethif =
  Log.debug (fun l -> l "bringing up %s" ethif);
  (try Tuntap.set_up_and_running ethif
   with e -> Log.err (fun l -> l "rawling: %a" Fmt.exn e));
  Lwt_rawlink.open_link ~filter:(bpf_filter ()) ethif

let check_exit_status cmd status =
  let cmds = String.concat " " cmd in
  match status with
  | Unix.WEXITED 0   -> Lwt.return_unit
  | Unix.WEXITED i   -> failf "%s: exit %d" cmds i
  | Unix.WSIGNALED i -> failf "%s: signal %d" cmds i
  | Unix.WSTOPPED i  -> failf "%s: stopped %d" cmds i

let parent cmd pid ethif =
  (* network traffic *)
  let rawlink = rawlink ethif in

  (* close child fds *)
  close_and_dup stdin  >>= fun () ->
  close (snd logs_out) >>= fun () ->
  close (snd logs_err) >>= fun () ->
  close (snd net)      >>= fun () ->
  close (snd store)    >>= fun () ->
  (*
  close (snd metrics) >>= fun () ->
  *)
  let wait () =
    Lwt_unix.waitpid [] pid >>= fun (_pid, w) ->
    Lwt_io.flush_all () >>= fun () ->

    check_exit_status cmd w
  in
  Lwt.pick [
    wait ();
    (* data *)
    proxy_rawlink ~rawlink ~fd:(fst net);

    (* redirect the calf stdout to the shim stdout *)
    forward ~src:(fst logs_out) ~dst:stdout;
    forward ~src:(fst logs_err) ~dst:stderr;
    (* metrics: TODO *)

    Store.serve ();
    Handlers.install ();
  ]

let run () cmd ethif =
  Lwt_main.run (
    Lwt_io.flush_all () >>= fun () ->
    match Lwt_unix.fork () with
    | 0   -> child cmd
    | pid -> parent cmd pid ethif
  )

(* CLI *)

open Cmdliner

let setup_log style_renderer level =
  Fmt_tty.setup_std_outputs ?style_renderer ();
  Logs.set_level level;
  let pp_header ppf x =
    Fmt.pf ppf "%5d: %a " (Unix.getpid ()) Logs_fmt.pp_header x
  in
  Logs.set_reporter (Logs_fmt.reporter ~pp_header ());
  ()

let setup_log =
  Term.(const setup_log $ Fmt_cli.style_renderer () $ Logs_cli.level ())

let cmd =
  (* FIXME: use runc isolation
   let default_cmd = [
    "/usr/bin/runc"; "--"; "run";
    "--bundle"; "/containers/images/000-dhcp-client";
    "dhcp-client"
  ] in
  *)
  let default_cmd = [
    "/dhcp-client-calf"; "--store=10"; "--net=12"
  ] in
  let doc =
    Arg.info ~docv:"CMD" ~doc:"Command to run the calf process." ["cmd"]
  in
  Arg.(value & opt (list ~sep:' ' string) default_cmd & doc)

let ethif =
  let doc =
    Arg.info ~docv:"NAME" ~doc:"The interface to listen too." ["ethif"]
  in
  Arg.(value & opt string "eth0" & doc)

let run =
  Term.(const run $ setup_log $ cmd $ ethif),
  Term.info "dhcp-client" ~version:"0.0"

let () = match Term.eval run with
  | `Error _ -> exit 1
  | _        -> exit 0

(*

let kv_store = Unix.pipe ()

let install_logger () =
  Logs_syslog_lwt.udp_reporter (Unix.inet_addr_of_string "127.0.0.1") ()
  >|= fun r ->
  Logs.set_reporter r

let () = Lwt_main.run (
    install_logger () >>= fun () ->
    fd_of_tap0 >>= fun fd ->
  )
*)
