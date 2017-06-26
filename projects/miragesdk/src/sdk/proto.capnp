@0x9e83562906de8259;

interface Flow {

  struct ReadResult {
    union {
      data  @0: Data;
      eof   @1: Void;
      error @2: Text;
    }
  }

  struct WriteResult {
    union {
      ok     @0: Void;
      closed @1: Void;
      error  @2: Text;
    }
  }

  read   @0 () -> ReadResult;
  write  @1 (buffer: Data) -> WriteResult;
  writev @2 (buffers: List(Data)) -> WriteResult;
  close  @3 () -> ();
}

interface Net {

  interface Callback {
    f @0 (buffer :Data) -> ();
  }

  struct Result {
    union {
      ok            @0: Void;
      disconnected  @1: Void;
      unimplemented @2: Void;
      error         @3: Text;
    }
  }

  disconnect @0 () -> ();
  write      @1 (buffer: Data) -> Result;
  writev     @2 (buffers: List(Data)) -> Result;
  listen     @3 (callback: Callback) -> Result;
  mac        @4 () -> (mac: Text); # FIXME: better type
}

# FIXME: replace ip and mac by proper types for Mac and IP adresses
interface Host {
  intf        @0 () -> (intf: Text);
  mac         @1 () -> (mac: Text);
  dhcpOptions @2 () -> (options: List(Text));
  setIp       @3 (ip: Text) -> ();
  setGateway  @4 (ip: Text) -> ();
}

interface Conf {

  interface Callback {
    f @0 (change :Data) -> ();
  }

  struct Response {
    union {
      ok       @0 :Data;
      notFound @1 :Void;
    }
  }

  write  @0 (path :List(Text), data: Data) -> ();
  read   @1 (path :List(Text)) -> Response;
  delete @2 (path :List(Text)) -> ();
  watch  @3 (path :List(Text), callback :Callback) -> ();
}
