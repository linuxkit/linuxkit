(* https://github.com/Engil/Canopy/blob/3b5573ad0be0fa729b6d4e1ca8b9bb348e164960/inflator.ml *)

let input_buffer = Bytes.create 0xFFFF
let output_buffer = Bytes.create 0xFFFF
let window = Decompress.Window.create ~proof:Decompress.B.proof_bytes

let deflate ?(level = 4) buff =
  let pos = ref 0 in
  let res = Buffer.create (Cstruct.len buff) in
  Decompress.Deflate.bytes input_buffer output_buffer
    (fun input_buffer -> function
       | Some _ ->
         let n = min 0xFFFF (Cstruct.len buff - !pos) in
         Cstruct.blit_to_bytes buff !pos input_buffer 0 n;
         pos := !pos + n;
         n
       | None ->
         let n = min 0xFFFF (Cstruct.len buff - !pos) in
         Cstruct.blit_to_bytes buff !pos input_buffer 0 n;
         pos := !pos + n;
         n)
    (fun output_buffer len ->
       Buffer.add_subbytes res output_buffer 0 len;
       0xFFFF)
    (Decompress.Deflate.default ~proof:Decompress.B.proof_bytes level)
  |> function
  | Ok _    -> Cstruct.of_string (Buffer.contents res)
  | Error _ -> failwith "Deflate.deflate"

let inflate ?output_size orig =
  let res = Buffer.create (match output_size with
      | Some len -> len
      | None -> Mstruct.length orig)
  in
  Decompress.Inflate.bytes input_buffer output_buffer
    (fun input_buffer ->
       let n = min 0xFFFF (Mstruct.length orig) in
       let s = Mstruct.get_string orig n in
       Bytes.blit_string s 0 input_buffer 0 n;
       n)
    (fun output_buffer len ->
       Buffer.add_subbytes res output_buffer 0 len;
       0xFFFF)
    (Decompress.Inflate.default (Decompress.Window.reset window))
  |> function
  | Ok _    -> Some (Mstruct.of_string (Buffer.contents res))
  | Error _ -> None
