
exception ReadClosed

let fusermount = "fusermount"
let path = "/Transfuse"

module Log = struct
  let fatal fmt =
    Printf.ksprintf (fun s -> prerr_endline s; exit 1) fmt
  let error fmt =
    Printf.ksprintf (fun s -> prerr_endline s) fmt
  let info fmt =
    Printf.ksprintf (fun s -> prerr_endline s) fmt
end

let events_path = path ^ "/events"

let try_fork caller_name =
  try
    Unix.fork ()
  with Unix.Unix_error (err,"fork",_) ->
    Log.fatal "%s fork failed: %s" caller_name (Unix.error_message err)

let check_status = function
  | Unix.WEXITED 0 -> Result.Ok ()
  | Unix.WEXITED k ->
    Result.Error ("exit code "^(string_of_int k))
  | Unix.WSIGNALED k ->
    Result.Error ("ocaml kill signal "^(string_of_int k))
  | Unix.WSTOPPED k ->
    Result.Error ("ocaml stop signal "^(string_of_int k))

let finally f at_end =
  let r = try f () with e -> (at_end (); raise e) in
  at_end ();
  r

let copy description dst src =
  let sz = 1 lsl 16 in
  let buf = Bytes.create sz in
  let pnum = ref 0 in
  let rec loop () =
    let n = Unix.read src buf 0 sz in
    (if n = 0 then raise ReadClosed);

    let fd = Unix.(
      openfile ("/tmp/"^description^"_"^(string_of_int !pnum))
        [O_WRONLY; O_CREAT] 0o600) in
    let k = Unix.write fd buf 0 n in
    assert (k = n);
    Unix.close fd;
    incr pnum;
    
    let written = Unix.write dst buf 0 n in
    (if n <> written
     then Log.error "copy of %s read %d but wrote %d" description n written);
    loop ()
  in
  try loop ()
  with
  | ReadClosed -> raise ReadClosed
  | e -> (Log.error "copy for %s failed" description; raise e)

let with_reader id f =
  let read_path = Printf.sprintf "%s/connections/%d/read" path id in
  let read_fd = Unix.(openfile read_path [O_RDONLY] 0o000) in
  try finally (fun () ->
    f read_path read_fd
  ) (fun () ->
    Unix.close read_fd
  ) with
  | ReadClosed -> exit 0

let with_writer id f =
  let write_path = Printf.sprintf "%s/connections/%d/write" path id in
  let write_fd = Unix.(openfile write_path [O_WRONLY] 0o000) in
  finally (fun () ->
    f write_path write_fd
  ) (fun () ->
    Unix.close write_fd;
    Unix.unlink write_path
  )

let read_opts id = with_reader id (fun read_path read_fd ->
  let sz = 512 in
  let buf = Bytes.create sz in
  let n = Unix.read read_fd buf 0 sz in
  let opts = Stringext.split ~on:'\000' (Bytes.sub buf 0 n) in
  Array.of_list opts
)

let get_fuse_sock opts =
  let wsock, rsock = Unix.(socketpair PF_UNIX SOCK_STREAM 0) in
  let wfd = Fd_send_recv.int_of_fd wsock in
  let opts = Array.append [|fusermount|] opts in
  let pid = Unix.(create_process_env fusermount opts
                    [|"_FUSE_COMMFD="^(string_of_int wfd)|]
                    stdin stdout stderr) in
  let _, status = Unix.waitpid [] pid in
  let () = Unix.(shutdown wsock SHUTDOWN_ALL) in
  match check_status status with
  | Result.Error str ->
    let opts = String.concat " " (Array.to_list opts) in
    Log.fatal "%s: %s" opts str
  | Result.Ok () ->
    (* We must read at least 1 byte, by POSIX! *)
    let _, _, fd = Fd_send_recv.recv_fd rsock "\000" 0 1 [] in
    let () = Unix.(shutdown rsock SHUTDOWN_ALL) in
    fd

(* readers fork into a new process *)
let start_reader id fuse =
  match try_fork "start_reader" with
  | 0 -> (* child *)
    with_reader id (fun _read_path read_fd ->
      copy ("reader_"^string_of_int id) fuse read_fd
    )
  | _child_pid -> (* parent *)
    ()

(* writers stay in the calling process *)
let start_writer id fuse = with_writer id (fun write_path write_fd ->
  copy ("writer_"^string_of_int id) write_fd fuse
)

let handle_connection id =
  Log.info "handle_connection %d" id;
  match try_fork "handle_connection" with
  | 0 -> (* child *)
    let opts = read_opts id in
    let fuse = get_fuse_sock opts in
    start_reader id fuse;
    begin try ignore (start_writer id fuse); exit 0
      with
      | e -> prerr_endline (Printexc.to_string e); exit 1
    end
  | _child_pid -> (* parent *)
    ()

let connection_loop () =
  let events = Unix.(openfile events_path [O_RDONLY] 0o000) in
  (* 512 bytes is easily big enough to read a whole connection id *)
  let sz = 512 in
  let buf = Bytes.create sz in
  let rec recv () =
    begin try
        let n = Unix.read events buf 0 sz in
        let s = String.trim Bytes.(to_string (sub buf 0 n)) in
        let id = int_of_string s in
        handle_connection id;
      with
      | Unix.Unix_error (err,"read",path) ->
        Log.fatal "Error reading events file %s: %s"
          path (Unix.error_message err)
      | Failure "int_of_string" ->
        Log.fatal "Failed to parse integer connection id"
    end;
    recv ()
  in
  recv ()

;;
connection_loop ()
