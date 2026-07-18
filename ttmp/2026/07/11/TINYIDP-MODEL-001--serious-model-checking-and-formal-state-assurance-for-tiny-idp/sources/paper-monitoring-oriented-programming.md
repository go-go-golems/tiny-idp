Research output: Contribution to journal › Article › peer-review

Monitoring-Oriented Programming (MOP <sup>1</sup>) \[21, 18, 22, 19\] is a formal framework for software development and analysis, in which the developer specifies desired properties using definable specification formalisms, along with code to execute when properties are violated or validated. The MOP framework automatically generates monitors from the specified properties and then integrates them together with the user-defined code into the original system. The previous design of MOP only allowed specifications without parameters, so it could not be used to state and monitor safety properties referring to two or more related objects. In this paper we propose a parametric specification-formalism-independent extension of MOP, together with an implementation of JavaMOP that supports parameters. In our current implementation, parametric specifications are translated into AspectJ code and then weaved into the application using off-the-shelf AspectJ compilers; hence, MOP specifications can be seen as formal or logical aspects. Our JavaMOP implementation was extensively evaluated on two benchmarks, Dacapo \[14\] and Tracematches \[8\], showing that runtime verification in general and MOP in particular are feasible. In some of the examples, millions of monitor instances are generated, each observing a set of related objects. To keep the runtime overhead of monitoring and event observation low, we devised and implemented a decentralized indexing optimization. Less than 8% of the experiments showed more than 10% runtime overhead; in most cases our tool generates monitoring code as efficient as the hand-optimized code. Despite its genericity, JavaMOP is empirically shown to be more efficient than runtime verification systems specialized and optimized for particular specification formalisms. Many property violations were detected during our experiments; some of them are benign, others indicate defects in programs. Many of these are subtle and hard to find by ordinary testing.

| Original language | English (US) |
| --- | --- |
| Pages (from-to) | 569-588 |
| Number of pages | 20 |
| Journal | [ACM SIGPLAN Notices](#) |
| Volume | 42 |
| Issue number | 10 |
| State | Published - |

- Aspect-oriented programming
- Monitoring-oriented programming
- Runtime verification

- General Computer Science

[Discover UIUC Full Text](https://i-share-uiu.primo.exlibrisgroup.com/openurl/01CARLI_UIU/01CARLI_UIU:CARLI_UIU?ctx_ver=Z39.88-2004&ctx_tim=2026-07-01T16%3A18%3A44UTC&ctx_enc=info%3Aofi%2FencUTF-8&url_ver=Z39.88-2004&url_ctx_fmt=info%3Aofi%2Ffmt%3Akev%3Amtx%3Actx&rft.genre=article&rft_val_fmt=info%3Aofi%2Fkev%3Afmt%3Ajournal&rfr_id=info%3Asid%2Fpure.atira.dk%3Apure&rft.atitle=MOP&rft.aulast=Chen&rft.aufirst=Feng&rft.auinit=F&rft.date=2007-10&rft.volume=42&rft.issue=10&rft.issn=1523-2867&rft.jtitle=ACM%20SIGPLAN%20Notices&rft.pages=569-588)

- [Link to publication in Scopus](https://www.scopus.com/pages/publications/51949108337)