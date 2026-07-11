## Abstract

We introduce a new programming language concept called typestate, which is a refinement of the concept of type. Whereas the type of a data object determines the set of operations ever permitted on the object, typestate determines the subset of these operations which is permitted in a particular context. Typestate tracking is a program analysis technique which enhances program reliability by detecting at compile-time syntactically legal but semantically undefined execution sequences. These include, for example, reading a variable before it has been initialized, dereferencing a pointer after the dynamic object has been deallocated, etc. Typestate tracking detects errors that cannot be detected by type checking or by conventional static scope rules. Additionally, typestate tracking makes it possible for compilers to insert appropriate finalization of data at exception points and on program termination, eliminating the need to support finalization by means of either garbage collection or unsafe deallocation operations such as Pascal’s dispose operation. By enforcing typestate invariants at compile-time, it becomes practical to implement a “secure language’’—that is, one in which all successfully compiled program modules have fully defined execution-time effects, and the only effects of program errors are incorrect output values. This paper defines typestate, gives examples of its application, and shows how typestate checking may be embedded into a compiler. We discuss the consequences of typestate checking for software reliability and software structure, and conclude with a discussion of our experience using a high-level language incorporating typestate checking. © 1986 IEEE

## Related

Conference paper

### NIL: An integrated language and system for distributed programming

Robert E. Strom, Shaula Yemini

Programming Language Issues in Software Systems 1983

Paper

### Interfaces, protocols, and the semi-automatic construction of software adaptors

Daniel M. Yellin, Robert E. Strom

ACM SIGPLAN Notices

Conference paper

### Interactive blackbox debugging for concurrent languages

German S. Goldszmidt, Shmuel Katz, et al.

WPADD 1988

David F. Bacon, Robert E. Strom, et al.

SIGPLAN Notices (ACM Special Interest Group on Programming Languages)