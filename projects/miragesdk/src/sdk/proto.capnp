@0x9e83562906de8259;

struct Response {
  union {
    ok       @0 :Data;
    notFound @1 :Void;
  }
}

interface Ctl {
  write  @0 (path :List(Text), data: Data) -> ();
  read   @1 (path :List(Text)) -> Response;
  delete @2 (path :List(Text)) -> ();
}
