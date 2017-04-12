open Lwt.Infix

let src = Logs.Src.create "init" ~doc:"Init steps"
module Log = (val Logs.src_log src : Logs.LOG)

let failf fmt = Fmt.kstrf Lwt.fail_with fmt

let pp_fd ppf (t:Lwt_unix.file_descr) =
  Fmt.int ppf (Obj.magic (Lwt_unix.unix_file_descr t): int)

let rec really_write fd buf off len =
  match len with
  | 0   -> Lwt.return_unit
  | len ->
    Log.debug (fun l -> l "really_write %a off=%d len=%d" pp_fd fd off len);
    Lwt_unix.write fd buf off len >>= fun n ->
    if n = 0 then Lwt.fail_with "write 0"
    else really_write fd buf (off+n) (len-n)

let write_all fd buf = really_write fd buf 0 (String.length buf)

let read_all fd =
  Log.debug (fun l -> l "read_all %a" pp_fd fd);
  let len = 16 * 1024 in
  let buf = Bytes.create len in
  let rec loop acc =
    Lwt_unix.read fd buf 0 len >>= fun n ->
    if n = 0 then failf "read %a: 0" pp_fd fd
    else
      let acc = String.sub buf 0 n :: acc in
      if n <= len then Lwt.return (List.rev acc)
      else loop acc
  in
  loop [] >|= fun bufs ->
  String.concat "" bufs

module Flow = struct

  (* build a flow from Lwt_unix.file_descr *)
  module Fd: Mirage_flow_lwt.CONCRETE with type flow = Lwt_unix.file_descr = struct
    type 'a io = 'a Lwt.t
    type buffer = Cstruct.t
    type error = [`Msg of string]
    type write_error = [ Mirage_flow.write_error | error ]

    let pp_error ppf (`Msg s) = Fmt.string ppf s

    let pp_write_error ppf = function
      | #Mirage_flow.write_error as e -> Mirage_flow.pp_write_error ppf e
      | #error as e                   -> pp_error ppf e

    type flow = Lwt_unix.file_descr

    let err e =  Lwt.return (Error (`Msg (Printexc.to_string e)))

    let read t =
      Lwt.catch (fun () ->
          read_all t >|= fun buf -> Ok (`Data (Cstruct.of_string buf))
        ) (function Failure _ -> Lwt.return (Ok `Eof) | e -> err e)

    let write t b =
      Lwt.catch (fun () ->
          write_all t (Cstruct.to_string b) >|= fun () -> Ok ()
        ) (fun e  -> err e)

    let close t = Lwt_unix.close t

    let writev t bs =
      Lwt.catch (fun () ->
          Lwt_list.iter_s (fun b -> write_all t (Cstruct.to_string b)) bs
          >|= fun () -> Ok ()
        ) (fun e -> err e)
  end

  (* build a flow from rawlink *)
  module Rawlink: Mirage_flow_lwt.CONCRETE with type flow = Lwt_rawlink.t = struct
    type 'a io = 'a Lwt.t
    type buffer = Cstruct.t
    type error = [`Msg of string]
    type write_error = [ Mirage_flow.write_error | error ]

    let pp_error ppf (`Msg s) = Fmt.string ppf s

    let pp_write_error ppf = function
      | #Mirage_flow.write_error as e -> Mirage_flow.pp_write_error ppf e
      | #error as e                   -> pp_error ppf e

    type flow = Lwt_rawlink.t

    let err e =  Lwt.return (Error (`Msg (Printexc.to_string e)))

    let read t =
      Lwt.catch (fun () ->
          Lwt_rawlink.read_packet t >|= fun buf -> Ok (`Data buf)
        ) (function Failure _ -> Lwt.return (Ok `Eof) | e -> err e)

    let write t b =
      Lwt.catch (fun () ->
          Lwt_rawlink.send_packet t b >|= fun () -> Ok ()
        ) (fun e  -> err e)

    let close t = Lwt_rawlink.close_link t

    let writev t bs =
      Lwt.catch (fun () ->
          Lwt_list.iter_s (Lwt_rawlink.send_packet t) bs >|= fun () -> Ok ()
        ) (fun e -> err e)
  end

  let int_of_fd t =
    (Obj.magic (Lwt_unix.unix_file_descr t): int)

  let fd ?name t =
    IO.create (module Fd) t (match name with
        | None   -> string_of_int (int_of_fd t)
        | Some n -> n)

end

let file_descr ?name t = Flow.fd ?name t

let rawlink ?filter ethif =
  Log.debug (fun l -> l "bringing up %s" ethif);
  (try Tuntap.set_up_and_running ethif
   with e -> Log.err (fun l -> l "rawlink: %a" Fmt.exn e));
  let t = Lwt_rawlink.open_link ?filter ethif in
  IO.create (module Flow.Rawlink) t ethif

module Fd = struct

  type t = {
    name: string;
    fd  : Lwt_unix.file_descr;
  }

  let stdout = { name = "stdout"; fd = Lwt_unix.stdout }
  let stderr = { name = "stderr"; fd = Lwt_unix.stderr }
  let stdin  = { name = "stdin" ; fd = Lwt_unix.stdin  }

  let of_int name (i:int) =
    let fd : Unix.file_descr = Obj.magic i in
    let fd = Lwt_unix.of_unix_file_descr fd in
    { name; fd }

  let to_int t =
    (Obj.magic (Lwt_unix.unix_file_descr t.fd): int)

  let pp ppf fd = Fmt.pf ppf "%s:%d" fd.name (to_int fd)

  let close fd =
    Log.debug (fun l -> l "close %a" pp fd);
    Unix.close (Lwt_unix.unix_file_descr fd.fd)

  let dev_null =
    Lwt_unix.of_unix_file_descr ~blocking:false
      (Unix.openfile "/dev/null" [Unix.O_RDWR] 0)

  let redirect_to_dev_null fd =
    Log.debug (fun l -> l "redirect-stdin-to-dev-null");
    Unix.close (Lwt_unix.unix_file_descr fd.fd);
    Lwt_unix.dup2 dev_null fd.fd;
    Unix.close (Lwt_unix.unix_file_descr dev_null)

  let dup2 ~src ~dst =
    Log.debug (fun l -> l "dup2 %a => %a" pp src pp dst);
    Lwt_unix.dup2 src.fd dst.fd;
    close src

  let flow t = Flow.fd t.fd ~name:(Fmt.to_to_string pp t)

end

module Pipe = struct

  type t = Fd.t * Fd.t

  type monitor = {
    stdout: t;
    stderr: t;
    metrics: t;
    ctl: t;
    net: t;
  }

  let stdout t = t.stdout
  let stderr t = t.stderr
  let metrics t = t.metrics
  let ctl t = t.ctl
  let net t = t.net

  let name (x, _) = x.Fd.name

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

  let v () =
    (* logs pipe *)
    let stdout = pipe "stdout" in
    let stderr = pipe "stderr" in
    (* network pipe *)
    let net = socketpair "net" in
    (* store pipe *)
    let ctl = socketpair "ctl" in
    (* metrics pipe *)
    let metrics = pipe "metrics" in
    { stdout; stderr; ctl; net; metrics }

end

let exec_calf t cmd =
  Log.info (fun l -> l "child pid is %d" Unix.(getpid ()));
  Fd.(redirect_to_dev_null stdin);

  (* close parent fds *)
  Fd.close Pipe.(priv t.stdout);
  Fd.close Pipe.(priv t.stderr);
  Fd.close Pipe.(priv t.ctl);
  Fd.close Pipe.(priv t.net);
  Fd.close Pipe.(priv t.metrics);

  let cmds = String.concat " " cmd in

  let calf_stdout = Fd.of_int "stdout" 1 in
  let calf_stderr = Fd.of_int "stderr" 2 in
  let calf_net    = Fd.of_int "net"    3 in
  let calf_ctl    = Fd.of_int "ctl"    4 in

  Log.info (fun l -> l "Executing %s" cmds);

  (* Move all open fds at the top *)
  Fd.dup2 ~src:Pipe.(calf t.net)    ~dst:calf_net;
  Fd.dup2 ~src:Pipe.(calf t.ctl)    ~dst:calf_ctl;
  Fd.dup2 ~src:Pipe.(calf t.stderr) ~dst:calf_stderr;
  Fd.dup2 ~src:Pipe.(calf t.stdout) ~dst:calf_stdout;

  (* exec the calf *)
  Unix.execve (List.hd cmd) (Array.of_list cmd) [||]

let check_exit_status cmd status =
  let cmds = String.concat " " cmd in
  match status with
  | Unix.WEXITED 0   -> Lwt.return_unit
  | Unix.WEXITED i   -> failf "%s: exit %d" cmds i
  | Unix.WSIGNALED i -> failf "%s: signal %d" cmds i
  | Unix.WSTOPPED i  -> failf "%s: stopped %d" cmds i

let exec_priv t ~pid cmd =

  Fd.(redirect_to_dev_null stdin);

  (* close child fds *)
  Fd.close Pipe.(calf t.stdout);
  Fd.close Pipe.(calf t.stderr);
  Fd.close Pipe.(calf t.net);
  Fd.close Pipe.(calf t.ctl);
  Fd.close Pipe.(calf t.metrics);

  let wait () =
    Lwt_unix.waitpid [] pid >>= fun (_pid, w) ->
    Lwt_io.flush_all () >>= fun () ->

    check_exit_status cmd w
  in
  Lwt.return wait

let block_for_ever =
  let t, _ = Lwt.task () in
  fun () -> t

let exec_and_forward ?(handlers=block_for_ever) ~pid ~cmd ~net ~ctl t =

  exec_priv t ~pid cmd >>= fun wait ->

  let priv_net    = Fd.flow Pipe.(priv t.net)    in
  let priv_ctl    = Fd.flow Pipe.(priv t.ctl)    in
  let priv_stdout = Fd.flow Pipe.(priv t.stdout) in
  let priv_stderr = Fd.flow Pipe.(priv t.stderr) in

  Lwt.pick ([
      wait ();
      (* data *)
      IO.proxy ~verbose:true net priv_net;

      (* redirect the calf stdout to the shim stdout *)
      IO.forward ~verbose:false ~src:priv_stdout ~dst:Fd.(flow stdout);
      IO.forward ~verbose:false ~src:priv_stderr ~dst:Fd.(flow stderr);
      (* TODO: Init.Fd.forward ~src:Init.Pipe.(priv metrics)
         ~dst:Init.Fd.metric; *)
      ctl priv_ctl;
      handlers ();
    ])

let exec t cmd fn =
  Lwt_io.flush_all () >>= fun () ->
  match Lwt_unix.fork () with
  | 0   -> exec_calf t cmd
  | pid -> fn pid

let run t ~net ~ctl ?handlers cmd =
  exec t cmd (fun pid -> exec_and_forward ?handlers ~pid ~cmd ~net ~ctl t)
