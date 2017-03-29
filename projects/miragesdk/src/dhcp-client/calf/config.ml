open Mirage

(* create a new device for mirage-net-fd *)
(* FIXME: should check it is invoked only with the unix backend *)
(* FIXME: this is a temporary solution, this should be exposed
   as a ukvm/virtio device  *)
let netif_of_fd id = impl @@
  let key = Key.abstract id in
  object
    inherit base_configurable
    method ty = network
    val name = Functoria_app.Name.create "net" ~prefix:"net"
    method name = name
    method module_name = "Netif_fd"
    method keys = [ key ]
    method packages = Key.pure [ package "mirage-net-fd" ]
    method connect _ modname _ =
      Fmt.strf "@[let (key: int) = %a in@,
                %s.connect (Obj.magic key: Unix.file_descr)@]"
        Key.serialize_call key modname
    method configure i =
      Ok ()
  end

let dhcp_codes =
  let doc = Key.Arg.info ~docv:"OPT" ~doc:"DHCP options." ["c";"codes"] in
  Key.(abstract @@ create "codes" Arg.(opt (list string) [] doc))

let net =
  let doc =
    Key.Arg.info ~docv:"FD" ~doc:"Network interface" ["net"]
  in
  let key = Key.(create "input" Arg.(opt int 3 doc)) in
  netif_of_fd key

let ctl =
  let doc =
    Key.Arg.info ~docv:"FD" ~doc:"Control interface" ["ctl"]
  in
  let key = Key.(create "output" Arg.(opt int 4 doc)) in
  netif_of_fd key

let keys = [dhcp_codes]

let packages = [
  package "jsonm";
  package "charrua-client";
  package "duration";
  package "charrua-client" ~sublibs:["mirage"];
  package "cohttp" ~sublibs:["lwt"]
]

let main =
  foreign ~keys ~packages "Unikernel.Main"
    (time @-> network @-> network @-> job)

let () = register "dhcp-client" [main $ default_time $ net $ ctl]
