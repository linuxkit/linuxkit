@0x9e83562906de8259;

struct Request {
  id   @0 :Int32;
  path @1 :List(Text);
  union {
    write  @2 :Data;
    read   @3 :Void;
    delete @4 :Void;
  }
}

struct Response {
  id   @0: Int32;
  union {
    ok    @1 :Data;
    error @2 :Data;
  }
}
