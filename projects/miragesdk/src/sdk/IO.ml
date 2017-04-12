open Lwt.Infix

let src = Logs.Src.create "IO" ~doc:"IO helpers"
module Log = (val Logs.src_log src : Logs.LOG)

(* from mirage-conduit. FIXME: move to mirage-flow *)
type 'a io = 'a Lwt.t
type buffer = Cstruct.t
type error = [`Msg of string]
type write_error = [ Mirage_flow.write_error | error ]
let pp_error ppf (`Msg s) = Fmt.string ppf s

let pp_write_error ppf = function
  | #Mirage_flow.write_error as e -> Mirage_flow.pp_write_error ppf e
  | #error as e                   -> pp_error ppf e

type flow =
  | Flow: string
          * (module Mirage_flow_lwt.CONCRETE with type flow = 'a)
          * 'a
    -> flow

let create (type a) (module M: Mirage_flow_lwt.S with type flow = a) t name =
  let m =
    (module Mirage_flow_lwt.Concrete(M):
       Mirage_flow_lwt.CONCRETE with type flow = a)
  in
  Flow (name, m , t)

let read (Flow (_, (module F), flow)) = F.read flow
let write (Flow (_, (module F), flow)) b = F.write flow b
let writev (Flow (_, (module F), flow)) b = F.writev flow b
let close (Flow (_, (module F), flow)) = F.close flow
let pp ppf (Flow (name, _, _)) = Fmt.string ppf name

type t = flow

let forward ?(verbose=false) ~src ~dst =
  let rec loop () =
    read src >>= function
    | Ok `Eof ->
      Log.err (fun l -> l "forward[%a => %a] EOF" pp src pp dst);
      Lwt.return_unit
    | Error e ->
      Log.err (fun l -> l "forward[%a => %a] %a" pp src pp dst pp_error e);
      Lwt.return_unit
    | Ok (`Data buf) ->
      Log.debug (fun l ->
          let payload =
            if verbose then Fmt.strf "[%S]" @@ Cstruct.to_string buf
            else Fmt.strf "%d bytes" (Cstruct.len buf)
          in
          l "forward[%a => %a] %s" pp src pp dst payload);
      write dst buf >>= function
      | Ok ()   -> loop ()
      | Error e ->
        Log.err (fun l -> l "forward[%a => %a] %a"
                    pp src pp dst pp_write_error e);
        Lwt.return_unit
  in
  loop ()

let proxy ?verbose f1 f2 =
  Lwt.join [
    forward ?verbose ~src:f1 ~dst:f2;
    forward ?verbose ~src:f2 ~dst:f1;
  ]
