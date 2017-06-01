@0xb224be3ea8450819;

struct DhcpNetworkRequest {
  id   @0 :Int32;
  path @1 :List(Text);
  union {
    write  @2 :Data;
    read   @3 :Void;
    delete @4 :Void;
  }
}

struct DhcpNetworkResponse {
  id   @0: Int32;
  union {
    ok    @1 :Data;
    error @2 :Data;
  }
}

struct DhcpActuatorRequest {
  id   @0 :Int32;
  interface @1 :Text;
  ipv4Addr @2 :List(Text);
  resolvConf @3 :List(Text);
}

struct DhcpActuatorResponse {
  id   @0: Int32;
  union {
    ok    @1 :Data;
    error @2 :Data;
  }
}

