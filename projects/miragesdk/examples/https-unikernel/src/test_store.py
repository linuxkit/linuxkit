import capnp
import capnp
import proto_capnp
import socket
s = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM, 0)
s.connect('/tmp/store.sock')
client = capnp.TwoPartyClient(s)
store = client.bootstrap().cast_as(proto_capnp.Store)
r = store.get(['index.html'])
print r.wait()
