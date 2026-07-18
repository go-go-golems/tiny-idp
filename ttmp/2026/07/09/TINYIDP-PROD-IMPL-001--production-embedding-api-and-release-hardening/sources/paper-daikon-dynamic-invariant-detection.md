**Download:** [PDF](https://homes.cs.washington.edu/~mernst/pubs/daikon-tool-scp2007.pdf), [Daikon implementation](https://plse.cs.washington.edu/daikon/).

“The Daikon system for dynamic detection of likely invariants” by [Michael D. Ernst](https://homes.cs.washington.edu/~mernst/), Jeff H. Perkins, [Philip J. Guo](https://pg.ucsd.edu/), [Stephen McCamant](https://people.csail.mit.edu/smcc/), [Carlos Pacheco](https://people.csail.mit.edu/cpacheco/), Matthew S. Tschantz, and Chen Xiao. *Science of Computer Programming*, vol. 69, no. 1--3, Dec. 2007, pp. 35-45.

## Abstract

Daikon is an implementation of dynamic detection of likely invariants; that is, the Daikon invariant detector reports likely program invariants. An invariant is a property that holds at a certain point or points in a program; these are often used in assert statements, documentation, and formal specifications. Examples include being constant (*x = a*), non-zero (*x ≠ 0*), being in a range (*a ≤ x ≤ b*), linear relationships (*y = ax+b*), ordering (*x ≤ y*), functions from a library (*x =* fn *(y)*), containment (*x ∈ y*), sortedness (*x* is sorted), and many more. Users can extend Daikon to check for additional invariants.

Dynamic invariant detection runs a program, observes the values that the program computes, and then reports properties that were true over the observed executions. Dynamic invariant detection is a machine learning technique that can be applied to arbitrary data. Daikon can detect invariants in C, C + +, Java, and Perl programs, and in record-structured data sources; it is easy to extend Daikon to other applications.

Invariants can be useful in program understanding and a host of other applications. Daikon's output has been used for generating test cases, predicting incompatibilities in component integration, automating theorem-proving, repairing inconsistent data structures, and checking the validity of data streams, among other tasks.

Daikon is freely available in source and binary form, along with extensive documentation, at [https://plse.cs.washington.edu/daikon/](https://plse.cs.washington.edu/daikon/).

**Download:** [PDF](https://homes.cs.washington.edu/~mernst/pubs/daikon-tool-scp2007.pdf), [Daikon implementation](https://plse.cs.washington.edu/daikon/).

**BibTeX entry:**

```
@article{ErnstPGMPTX2007,
   author = {Michael D. Ernst and Jeff H. Perkins and Philip J. Guo and
    Stephen McCamant and Carlos Pacheco and Matthew S. Tschantz and
    Chen Xiao},
   title = {The {Daikon} system for dynamic detection of likely invariants},
   journal = {Science of Computer Programming},
   volume = {69},
   number = {1--3},
   pages = {35--45},
   month = dec,
   year = {2007}
}
```

---

(This webpage was created with [bibtex2web](https://homes.cs.washington.edu/~mernst/software/#bibtex2web).)