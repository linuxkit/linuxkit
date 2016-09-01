(* ocamlfind ocamlopt -package unix,astring -linkpkg -o iptables iptables.ml *)

(*
--wait -t nat -I DOCKER-INGRESS -p tcp --dport 80 -j DNAT --to-destination 172.18.0.2:80
--wait -t nat -D DOCKER-INGRESS -p tcp --dport 80 -j DNAT --to-destination 172.18.0.2:80
*)

let _iptables = "/sbin/iptables"
let _proxy = "/usr/bin/docker-proxy"
let _pid_dir = "/var/run/service-port-opener"

type port = {
  proto: string;
  dport: string; (* host port *)
  ip:    string; (* container ip *)
  port:  string; (* container port *)
}

let syslog = Syslog.openlog ~facility:`LOG_SECURITY "iptables-wrapper"

let logf fmt =
  Printf.ksprintf (fun s ->
    Syslog.syslog syslog `LOG_INFO s
  ) fmt

let pid_filename { proto; dport; ip; port } =
  Printf.sprintf "%s/%s.%s.%s.%s.pid" _pid_dir proto dport ip port

let insert ({ proto; dport; ip; port } as p) =
  let filename = pid_filename p in
  logf "insert: creating a proxy for %s" filename;
  let args = [ _proxy; "-proto"; proto; "-container-ip"; ip; "-container-port"; port; "-host-ip"; "0.0.0.0"; "-host-port"; dport; "-i"; "-no-local-ip" ] in
  let pid = Unix.fork () in
  if pid == 0 then begin
    logf "binary = %s args = %s" _proxy (String.concat "; " args);
    (* Close the vast number of fds I've inherited from docker *)
    (* TODO(djs55): revisit, possibly by filing a docker/docker issue *)
    for i = 0 to 1023 do
      let fd : Unix.file_descr = Obj.magic i in
      try Unix.close fd with Unix.Unix_error(Unix.EBADF, _, _) -> ()
    done;
    let null = Unix.openfile "/dev/null" [ Unix.O_RDWR ] 0 in
    Unix.dup2 null Unix.stdin;
    Unix.dup2 null Unix.stdout;
    Unix.dup2 null Unix.stderr;
    (try Unix.execv _proxy (Array.of_list args) with e -> logf "Failed with %s" (Printexc.to_string e));
    exit 1
  end else begin
    (* write pid to a file (not atomically) *)
    let oc = open_out filename in
    output_string oc (string_of_int pid);
    close_out oc
  end

let delete ({ proto; dport; ip; port } as p) =
  let filename = pid_filename p in
  logf "delete: removing a proxy for %s" filename;
  (* read the pid from a file *)
  try
    let ic = open_in filename in
    let pid = int_of_string (input_line ic) in
    logf "Sending SIGTERM to %d" pid;
    Unix.kill pid Sys.sigterm;
    Unix.unlink filename
  with e ->
    logf "delete: failed to remove proxy for %s: %s" filename (Printexc.to_string e);
    ()

let parse_ip_port ip_port = match Astring.String.cut ~sep:":" ip_port with
  | None ->
    failwith ("Failed to parse <ip:port>:" ^ ip_port)
  | Some (ip, port) ->
    ip, port

let _ =
  ( try Unix.mkdir _pid_dir 0o0755 with Unix.Unix_error(Unix.EEXIST, _, _) -> () );
  let port_forwarding =
    try
      let ic = open_in "/Database/branch/master/ro/com.docker.driver.amd64-linux/native/port-forwarding" in
      bool_of_string (String.trim (input_line ic))
    with _ -> false in
  logf "port_forwarding=%b intercepted arguments [%s]" port_forwarding (String.concat "; " (Array.to_list Sys.argv));
  if port_forwarding then begin
    match Array.to_list Sys.argv with
    | [ _; "--wait"; "-t"; "nat"; "-I"; "DOCKER-INGRESS"; "-p"; proto; "--dport"; dport; "-j"; "DNAT"; "--to-destination"; ip_port ] ->
      let ip, port = parse_ip_port ip_port in
      insert { proto; dport; ip; port }
    | [ _; "--wait"; "-t"; "nat"; "-D"; "DOCKER-INGRESS"; "-p"; proto; "--dport"; dport; "-j"; "DNAT"; "--to-destination"; ip_port ] ->
      let ip, port = parse_ip_port ip_port in
      delete { proto; dport; ip; port }
    | _ ->
      ()
  end;
  Unix.execv _iptables Sys.argv
