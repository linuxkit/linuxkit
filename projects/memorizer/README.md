# Memorizer

Memorizer is a tool to trace fine-grained intra-kernel
operations. The goal is to track interactions with memory
objects for the purpose of analyzing fine-grained
interactions amongst components and execution contexts.
Memorizer tracks the following object operations: creation
(alloc), destruction (free), modify (store), access (load),
call, and return. 
