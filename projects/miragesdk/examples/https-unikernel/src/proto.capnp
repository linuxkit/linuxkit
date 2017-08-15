@0xe81d238ec50a0daa;

interface Store {
  struct GetResults {
    union {
      ok @0 :Text;
      notFound @1 :Void;
    }
  }

  get @0 (path :List(Text)) -> GetResults;
}

interface Flow {
  read @0 () -> (data :Data); 	# "" means end-of-file
  write @1 (data :Data) -> ();
}

interface HttpServer {
  accept @0 (connection :Flow) -> ();
}
