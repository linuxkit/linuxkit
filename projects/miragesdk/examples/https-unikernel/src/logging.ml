module Actor = struct
  type t = Fmt.style * string
  let pp f (style, name) = Fmt.(styled style (const string name)) f ()
  let tag = Logs.Tag.def "actor" pp
end

let pp_qid f = function
  | None -> ()
  | Some x ->
    let s = Uint32.to_string x in
    Fmt.(styled `Magenta (fun f x -> Fmt.pf f " (qid=%s)" x)) f s

let reporter =
  let report src level ~over k msgf =
    let src = Logs.Src.name src in
    msgf @@ fun ?header ?(tags=Logs.Tag.empty) fmt ->
    let actor =
      match Logs.Tag.find Actor.tag tags with
      | Some x -> x
      | None -> `Black, "------"
    in
    let qid = Logs.Tag.find Capnp_rpc.Debug.qid_tag tags in
    let print _ =
      Fmt.(pf stderr) "%a@." pp_qid qid;
      over ();
      k ()
    in
    Fmt.kpf print Fmt.stderr ("%a %a %a: @[" ^^ fmt ^^ "@]")
      Fmt.(styled `Magenta string) (Printf.sprintf "%11s" src)
      Logs_fmt.pp_header (level, header)
      Actor.pp actor
  in
  { Logs.report = report }

let init () =
  Fmt_tty.setup_std_outputs ();
  Logs.set_reporter reporter;
  Logs.set_level (Some Logs.Info)
