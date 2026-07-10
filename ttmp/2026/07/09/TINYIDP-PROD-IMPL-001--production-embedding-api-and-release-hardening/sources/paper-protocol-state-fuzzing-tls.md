Joeri de Ruiter, *University of Birmingham;* Erik Poll, *Radboud University Nijmegen*

We describe a largely automated and systematic analysis of TLS implementations by what we call ‘protocol state fuzzing’: we use state machine learning to infer state machines from protocol implementations, using only blackbox testing, and then inspect the inferred state machines to look for spurious behaviour which might be an indication of flaws in the program logic. For detecting the presence of spurious behaviour the approach is almost fully automatic: we automatically obtain state machines and any spurious behaviour is then trivial to see. Detecting whether the spurious behaviour introduces exploitable security weaknesses does require manual investigation. Still, we take the point of view that any spurious functionality in a security protocol implementation is dangerous and should be removed.

We analysed both server- and client-side implementations with a test harness that supports several key exchange algorithms and the option of client certificate authentication. We show that this approach can catch an interesting class of implementation flaws that is apparently common in security protocol implementations: in three of the TLS implementations analysed new security flaws were found (in GnuTLS, the Java Secure Socket Extension, and OpenSSL). This shows that protocol state fuzzing is a useful technique to systematically analyse security protocol implementations. As our analysis of different TLS implementations resulted in different and unique state machines for each one, the technique can also be used for fingerprinting TLS implementations.

## Joeri de Ruiter, University of Birmingham

## Erik Poll, Radboud University Nijmegen

## Open Access Media

USENIX is committed to Open Access to the research presented at our events. Papers and proceedings are freely available to everyone once the event begins. Any video, audio, and/or slides that are posted after the event are also free and open to everyone. [Support USENIX](https://www.usenix.org/annual-fund) and our commitment to Open Access.

BibTeX

@inproceedings {190892,  
author = {Joeri de Ruiter and Erik Poll},  
title = {Protocol State Fuzzing of {TLS} Implementations},  
booktitle = {24th USENIX Security Symposium (USENIX Security 15)},  
year = {2015},  
isbn = {978-1-939133-11-3},  
address = {Washington, D.C.},  
pages = {193--206},  
url = {https://www.usenix.org/conference/usenixsecurity15/technical-sessions/presentation/de-ruiter},  
publisher = {USENIX Association},  
month = aug  
}