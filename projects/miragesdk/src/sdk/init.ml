open Lwt.Infix

let src = Logs.Src.create "init" ~doc:"Init steps"
module Log = (val Logs.src_log src : Logs.LOG)

let failf fmt = Fmt.kstrf Lwt.fail_with fmt

module IO = struct

  let rec really_write fd buf off len =
    match len with
    | 0   -> Lwt.return_unit
    | len ->
      Lwt_unix.write fd buf off len >>= fun n ->
      really_write fd buf (off+n) (len-n)

  let rec really_read fd buf off len =
    match len with
    | 0   -> Lwt.return_unit
    | len ->
      Lwt_unix.read fd buf off len >>= fun n ->
      really_write fd buf (off+n) (len-n)

  let read_all fd =
    let len = 16 * 1024 in
    let buf = Bytes.create len in
    let rec loop acc =
      Lwt_unix.read fd buf 0 len >>= fun len ->
      let res = String.sub buf 0 len in
      loop (res :: acc)
    in
    loop [] >|= fun bufs ->
    String.concat "" (List.rev bufs)

end

module Fd = struct

  type t = {
    name: string;
    fd  : Lwt_unix.file_descr;
  }

  let fd t = t.fd
  let stdout = { name = "stdout"; fd = Lwt_unix.stdout }
  let stderr = { name = "stderr"; fd = Lwt_unix.stderr }
  let stdin  = { name = "stdin" ; fd = Lwt_unix.stdin  }

  let to_int t =
    (Obj.magic (Lwt_unix.unix_file_descr t.fd): int)

  let pp ppf fd = Fmt.pf ppf "%s:%d" fd.name (to_int fd)

  let close fd =
    Log.debug (fun l -> l "close %a" pp fd);
    Lwt_unix.close fd.fd

  let dev_null =
    Lwt_unix.of_unix_file_descr ~blocking:false
      (Unix.openfile "/dev/null" [Unix.O_RDWR] 0)

  let redirect_to_dev_null fd =
    Log.debug (fun l -> l "redirect-stdin-to-dev-null");
    Lwt_unix.close fd.fd >>= fun () ->
    Lwt_unix.dup2 dev_null fd.fd;
    Lwt_unix.close dev_null

  let dup2 ~src ~dst =
    Log.debug (fun l -> l "dup2 %a => %a" pp src pp dst);
    Lwt_unix.dup2 src.fd dst.fd;
    close src

  let proxy_net ~net fd =
    Log.debug (fun l -> l "proxy-net eth0 <=> %a" pp fd);
    let rec listen_rawlink () =
      Lwt_rawlink.read_packet net >>= fun buf ->
      Log.debug (fun l -> l "PROXY-NET: => %a" Cstruct.hexdump_pp buf);
      Log.debug (fun l -> l "PROXY-NET: => %S" (Cstruct.to_string buf));
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
        Log.debug (fun l -> l "PROXY-NET: <= %a" Cstruct.hexdump_pp buf);
        Lwt_rawlink.send_packet net buf >>= fun () ->
        loop ()
      in
      loop ()
    in
    Lwt.pick [
      listen_rawlink ();
      listen_socket ();
    ]

  let forward ~src ~dst =
    Log.debug (fun l -> l "forward %a => %a" pp src pp dst);
    let len = 16 * 1024 in
    let buf = Bytes.create len in
    let rec loop () =
      Lwt_unix.read src.fd buf 0 len >>= fun len ->
      if len = 0 then
        (* FIXME: why this ever happen *)
        Fmt.kstrf Lwt.fail_with "FORWARD[%a => %a]: EOF" pp src pp dst
      else (
        Log.debug (fun l ->
            l "FORWARD[%a => %a]: %S (%d)"
              pp src pp dst (Bytes.sub buf 0 len) len);
        IO.really_write dst.fd buf 0 len >>= fun () ->
        loop ()
      )
    in
    loop ()

  let proxy x y =
    Lwt.pick [
      forward ~src:x ~dst:y;
      forward ~src:y ~dst:x;
    ]

end

module Pipe = struct

  type t = Fd.t * Fd.t

  let priv = fst
  let calf = snd

  let socketpair name =
    let priv, calf = Lwt_unix.(socketpair PF_UNIX SOCK_STREAM 0) in
    Lwt_unix.clear_close_on_exec priv;
    Lwt_unix.clear_close_on_exec calf;
    { Fd.name = name; fd = priv }, { Fd.name = name ^ "-calf"; fd = calf }

  let pipe name =
    let priv, calf = Lwt_unix.pipe () in
    Lwt_unix.clear_close_on_exec priv;
    Lwt_unix.clear_close_on_exec calf;
    { Fd.name = name; fd = priv }, { Fd.name = name ^ "-calf"; fd = calf }

  (* logs pipe *)
  let stdout = pipe "stdout"
  let stderr = pipe "stderr"

  (* store pipe *)
  let ctl = socketpair "ctl"

  (* network pipe *)
  let net = socketpair "net"

  (* metrics pipe *)
  let metrics = pipe "metrics"

end

let exec_calf cmd =
  Fd.(redirect_to_dev_null stdin) >>= fun () ->

  (* close parent fds *)
  Fd.close Pipe.(priv stdout)  >>= fun () ->
  Fd.close Pipe.(priv stderr)  >>= fun () ->
  Fd.close Pipe.(priv ctl)     >>= fun () ->
  Fd.close Pipe.(priv net)     >>= fun () ->
  Fd.close Pipe.(priv metrics) >>= fun () ->

  let cmds = String.concat " " cmd in

  let calf_net = Pipe.(calf net) in
  let calf_ctl = Pipe.(calf ctl) in
  let calf_stdout = Pipe.(calf stdout) in
  let calf_stderr = Pipe.(calf stderr) in

  Log.info (fun l -> l "Executing %s" cmds);
  Log.debug (fun l -> l "net-fd=%a store-fd=%a" Fd.pp calf_net Fd.pp calf_ctl);

  Fd.dup2 ~src:calf_stdout ~dst:Fd.stdout >>= fun () ->
  Fd.dup2 ~src:calf_stderr ~dst:Fd.stderr >>= fun () ->

  (* exec the calf *)
  Unix.execve (List.hd cmd) (Array.of_list cmd) [||]

let rawlink ?filter ethif =
  Log.debug (fun l -> l "bringing up %s" ethif);
  (try Tuntap.set_up_and_running ethif
   with e -> Log.err (fun l -> l "rawlink: %a" Fmt.exn e));
  Lwt_rawlink.open_link ?filter ethif

let check_exit_status cmd status =
  let cmds = String.concat " " cmd in
  match status with
  | Unix.WEXITED 0   -> Lwt.return_unit
  | Unix.WEXITED i   -> failf "%s: exit %d" cmds i
  | Unix.WSIGNALED i -> failf "%s: signal %d" cmds i
  | Unix.WSTOPPED i  -> failf "%s: stopped %d" cmds i

let exec_priv ~pid ~cmd ~net ~ctl ~handlers =

  Fd.(redirect_to_dev_null stdin) >>= fun () ->

  (* close child fds *)
  Fd.close Pipe.(calf stdout)  >>= fun () ->
  Fd.close Pipe.(calf stderr)  >>= fun () ->
  Fd.close Pipe.(calf net)     >>= fun () ->
  Fd.close Pipe.(calf ctl)     >>= fun () ->
  Fd.close Pipe.(calf metrics) >>= fun () ->

  let wait () =
    Lwt_unix.waitpid [] pid >>= fun (_pid, w) ->
    Lwt_io.flush_all () >>= fun () ->

    check_exit_status cmd w
  in
  Lwt.pick ([
      wait ();
      (* data *)
      Fd.proxy_net ~net Pipe.(priv net);

      (* redirect the calf stdout to the shim stdout *)
      Fd.forward ~src:Pipe.(priv stdout)  ~dst:Fd.stdout;
      Fd.forward ~src:Pipe.(priv stderr)  ~dst:Fd.stderr;
      (* TODO: Init.Fd.forward ~src:Init.Pipe.(priv metrics) ~dst:Init.Fd.metric; *)
      ctl ();
(*      handlers (); *)
    ])

let run ~net ~ctl ~handlers cmd =
  Lwt_io.flush_all () >>= fun () ->
  match Lwt_unix.fork () with
  | 0   -> exec_calf cmd
  | pid -> exec_priv ~pid ~cmd ~net ~ctl ~handlers
