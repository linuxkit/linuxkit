type ro = Capnp.Message.ro
type rw = Capnp.Message.rw

module type S = sig
  type 'cap message_t

  type reader_t_Request_14112192289179464829
  type builder_t_Request_14112192289179464829
  type reader_t_Response_16897334327181152309
  type builder_t_Response_16897334327181152309

  module Reader : sig
    type array_t
    type builder_array_t
    type pointer_t
    module Response : sig
      type t = reader_t_Response_16897334327181152309
      type builder_t = builder_t_Response_16897334327181152309
      type unnamed_union_t =
        | Ok of string
        | Error of string
        | Undefined of int
      val get : t -> unnamed_union_t
      val id_get : t -> int32
      val id_get_int_exn : t -> int
      val of_message : 'cap message_t -> t
      val of_builder : builder_t -> t
    end
    module Request : sig
      type t = reader_t_Request_14112192289179464829
      type builder_t = builder_t_Request_14112192289179464829
      type unnamed_union_t =
        | Write of string
        | Read
        | Delete
        | Undefined of int
      val get : t -> unnamed_union_t
      val id_get : t -> int32
      val id_get_int_exn : t -> int
      val has_path : t -> bool
      val path_get : t -> (ro, string, array_t) Capnp.Array.t
      val path_get_list : t -> string list
      val path_get_array : t -> string array
      val of_message : 'cap message_t -> t
      val of_builder : builder_t -> t
    end
  end

  module Builder : sig
    type array_t = Reader.builder_array_t
    type reader_array_t = Reader.array_t
    type pointer_t
    module Response : sig
      type t = builder_t_Response_16897334327181152309
      type reader_t = reader_t_Response_16897334327181152309
      type unnamed_union_t =
        | Ok of string
        | Error of string
        | Undefined of int
      val get : t -> unnamed_union_t
      val ok_set : t -> string -> unit
      val error_set : t -> string -> unit
      val id_get : t -> int32
      val id_get_int_exn : t -> int
      val id_set : t -> int32 -> unit
      val id_set_int_exn : t -> int -> unit
      val of_message : rw message_t -> t
      val to_message : t -> rw message_t
      val to_reader : t -> reader_t
      val init_root : ?message_size:int -> unit -> t
    end
    module Request : sig
      type t = builder_t_Request_14112192289179464829
      type reader_t = reader_t_Request_14112192289179464829
      type unnamed_union_t =
        | Write of string
        | Read
        | Delete
        | Undefined of int
      val get : t -> unnamed_union_t
      val write_set : t -> string -> unit
      val read_set : t -> unit
      val delete_set : t -> unit
      val id_get : t -> int32
      val id_get_int_exn : t -> int
      val id_set : t -> int32 -> unit
      val id_set_int_exn : t -> int -> unit
      val has_path : t -> bool
      val path_get : t -> (rw, string, array_t) Capnp.Array.t
      val path_get_list : t -> string list
      val path_get_array : t -> string array
      val path_set : t -> (rw, string, array_t) Capnp.Array.t -> (rw, string, array_t) Capnp.Array.t
      val path_set_list : t -> string list -> (rw, string, array_t) Capnp.Array.t
      val path_set_array : t -> string array -> (rw, string, array_t) Capnp.Array.t
      val path_init : t -> int -> (rw, string, array_t) Capnp.Array.t
      val of_message : rw message_t -> t
      val to_message : t -> rw message_t
      val to_reader : t -> reader_t
      val init_root : ?message_size:int -> unit -> t
    end
  end
end

module Make (MessageWrapper : Capnp.MessageSig.S) :
  (S with type 'cap message_t = 'cap MessageWrapper.Message.t
    and type Reader.pointer_t = ro MessageWrapper.Slice.t option
    and type Builder.pointer_t = rw MessageWrapper.Slice.t
)

