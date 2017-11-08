/*
Package p9p implements a compliant 9P2000 client and server library for use
in modern, production Go services. This package differentiates itself in that
is has departed from the plan 9 implementation primitives and better follows
idiomatic Go style.

The package revolves around the session type, which is an enumeration of raw
9p message calls. A few calls, such as flush and version, have been elided,
defering their usage to the server implementation. Sessions can be trivially
proxied through clients and servers.

Getting Started

The best place to get started is with Serve. Serve can be provided a
connection and a handler. A typical implementation will call Serve as part of
a listen/accept loop. As each network connection is created, Serve can be
called with a handler for the specific connection. The handler can be
implemented with a Session via the Dispatch function or can generate sessions
for dispatch in response to client messages. (See cmd/9ps for an example)

On the client side, NewSession provides a 9p session from a connection. After
a version negotiation, methods can be called on the session, in parallel, and
calls will be sent over the connection. Call timeouts can be controlled via
the context provided to each method call.

Framework

This package has the beginning of a nice client-server framework for working
with 9p. Some of the abstractions aren't entirely fleshed out, but most of
this can center around the Handler.

Missing from this are a number of tools for implementing 9p servers. The most
glaring are directory read and walk helpers. Other, more complex additions
might be a system to manage in memory filesystem trees that expose multi-user
sessions.

Differences

The largest difference between this package and other 9p packages is
simplification of the types needed to implement a server. To avoid confusing
bugs and odd behavior, the components are separated by each level of the
protocol. One example is that requests and responses are separated and they no
longer hold mutable state. This means that framing, transport management,
encoding, and dispatching are componentized. Little work will be required to
swap out encodings, transports or connection implementations.

Context Integration

This package has been wired from top to bottom to support context-based
resource management. Everything from startup to shutdown can have timeouts
using contexts. Not all close methods are fully in place, but we are very
close to having controlled, predictable cleanup for both servers and clients.
Timeouts can be very granular or very course, depending on the context of the
timeout. For example, it is very easy to set a short timeout for a stat call
but a long timeout for reading data.

Multiversion Support

Currently, there is not multiversion support. The hooks and functionality are
in place to add multi-version support. Generally, the correct space to do this
is in the codec. Types, such as Dir, simply need to be extended to support the
possibility of extra fields.

The real question to ask here is what is the role of the version number in the
9p protocol. It really comes down to the level of support required. Do we just
need it at the protocol level, or do handlers and sessions need to be have
differently based on negotiated versions?

Caveats

This package has a number of TODOs to make it easier to use. Most of the
existing code provides a solid base to work from. Don't be discouraged by the
sawdust.

In addition, the testing is embarassingly lacking. With time, we can get full
testing going and ensure we have confidence in the implementation.
*/
package p9p
