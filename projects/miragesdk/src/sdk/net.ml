(* This file is a big hack and should be replaced ASAP with proper bindings *)

open Lwt.Infix

let run fmt =
  Fmt.kstrf (fun str ->
      match Sys.command str with
      | 0 -> Lwt.return ()
      | i -> Fmt.kstrf Lwt.fail_with "%S exited with code %d" str i
    ) fmt

let read fmt =
  Fmt.kstrf (fun str ->
      Lwt_process.pread ("/bin/sh", [|"/bin/sh"; "-c";  str|])
    ) fmt

let mac ethif =
  read "ifconfig -a %s | grep -o -E '([[:xdigit:]]{1,2}:){5}[[:xdigit:]]{1,2}'"
    ethif >|= fun mac ->
  Macaddr.of_string_exn (String.trim mac)

let set_ip ethif ip =
  (* FIXME: use language bindings to netlink instead *)
  (* run "ip addr add %s/24 dev %s" ip ethif *)
  run "ifconfig %s %a netmask 255.255.255.0" ethif Ipaddr.V4.pp_hum ip

let set_gateway gw =
  run "ip route add default via %a" Ipaddr.V4.pp_hum gw
