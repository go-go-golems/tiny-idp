**Download:** [PDF](https://homes.cs.washington.edu/~mernst/pubs/invariants-verify-rv2001.pdf).

“Static verification of dynamically detected program invariants: Integrating Daikon and ESC/Java” by Jeremy W. Nimmer and [Michael D. Ernst](https://homes.cs.washington.edu/~mernst/). In *RV 2001: Proceedings of the First Workshop on Runtime Verification*, (Paris, France), July 2001.

## Abstract

This paper shows how to integrate two complementary techniques for manipulating program invariants: dynamic detection and static verification. Dynamic detection proposes likely invariants based on program executions, but the resulting properties are not guaranteed to be true over all possible executions. Static verification checks that properties are always true, but it can be difficult and tedious to select a goal and to annotate programs for input to a static checker. Combining these techniques overcomes the weaknesses of each: dynamically detected invariants can annotate a program or provide goals for static verification, and static verification can confirm properties proposed by a dynamic tool.

We have integrated a tool for dynamically detecting likely program invariants, Daikon, with a tool for statically verifying program properties, ESC/Java. Daikon examines run-time values of program variables; it looks for patterns and relationships in those values, and it reports properties that are never falsified during test runs and that satisfy certain other conditions, such as being statistically justified. ESC/Java takes as input a Java program annotated with preconditions, postconditions, and other assertions, and it reports which annotations cannot be statically verified and also warns of potential runtime errors, such as null dereferences and out-of-bounds array indices.

Our prototype system runs Daikon, inserts its output into code as ESC/Java annotations, and then runs ESC/Java, which reports unverifiable annotations. The entire process is completely automatic, though users may provide guidance in order to improve results if desired. In preliminary experiments, ESC/Java verified all or most of the invariants proposed by Daikon.

**Download:** [PDF](https://homes.cs.washington.edu/~mernst/pubs/invariants-verify-rv2001.pdf).

**BibTeX entry:**

```
@inproceedings{NimmerE01:RV,
   author = {Jeremy W. Nimmer and Michael D. Ernst},
   title = {Static verification of dynamically detected program
    invariants: Integrating {Daikon} and {ESC}/{Java}},
   booktitle = {RV 2001: Proceedings of the First Workshop on Runtime
    Verification},
   address = {Paris, France},
   month = jul,
   year = {2001}
}
```

---

(This webpage was created with [bibtex2web](https://homes.cs.washington.edu/~mernst/software/#bibtex2web).)