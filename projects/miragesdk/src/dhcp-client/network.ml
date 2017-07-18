open Lwt.Infix

external dhcp_filter: unit -> string = "bpf_filter"

module Make (Act: Sdk.Host.S) = struct

  include Sdk.Net.Rawlink

  let connect act =
    let filter = dhcp_filter () in
    Act.mac act >>= fun mac ->
    Act.interface act >>= fun intf ->
    Sdk.Net.Rawlink.connect ~filter ~mac intf

end
