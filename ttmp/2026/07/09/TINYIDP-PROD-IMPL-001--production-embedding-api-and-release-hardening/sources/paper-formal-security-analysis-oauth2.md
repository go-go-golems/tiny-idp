## Title:A Comprehensive Formal Security Analysis of OAuth 2.0

Authors:[Daniel Fett](https://arxiv.org/search/cs?searchtype=author&query=Fett,+D), [Ralf Kuesters](https://arxiv.org/search/cs?searchtype=author&query=Kuesters,+R), [Guido Schmitz](https://arxiv.org/search/cs?searchtype=author&query=Schmitz,+G)

[View PDF](https://arxiv.org/pdf/1601.01229)

> Abstract:The OAuth 2.0 protocol is one of the most widely deployed authorization/single sign-on (SSO) protocols and also serves as the foundation for the new SSO standard OpenID Connect. Despite the popularity of OAuth, so far analysis efforts were mostly targeted at finding bugs in specific implementations and were based on formal models which abstract from many web features or did not provide a formal treatment at all.  
> In this paper, we carry out the first extensive formal analysis of the OAuth 2.0 standard in an expressive web model. Our analysis aims at establishing strong authorization, authentication, and session integrity guarantees, for which we provide formal definitions. In our formal analysis, all four OAuth grant types (authorization code grant, implicit grant, resource owner password credentials grant, and the client credentials grant) are covered. They may even run simultaneously in the same and different relying parties and identity providers, where malicious relying parties, identity providers, and browsers are considered as well. Our modeling and analysis of the OAuth 2.0 standard assumes that security recommendations and best practices are followed, in order to avoid obvious and known attacks.  
> When proving the security of OAuth in our model, we discovered four attacks which break the security of OAuth. The vulnerabilities can be exploited in practice and are present also in OpenID Connect.  
> We propose fixes for the identified vulnerabilities, and then, for the first time, actually prove the security of OAuth in an expressive web model. In particular, we show that the fixed version of OAuth (with security recommendations and best practices in place) provides the authorization, authentication, and session integrity properties we specify.

| Comments: |
| --- |
| Subjects: | Cryptography and Security (cs.CR) |
| Cite as: | [arXiv:1601.01229](https://arxiv.org/abs/1601.01229) \[cs.CR\] |
|  | (or [arXiv:1601.01229v4](https://arxiv.org/abs/1601.01229v4) \[cs.CR\] for this version) |
|  | [https://doi.org/10.48550/arXiv.1601.01229](https://doi.org/10.48550/arXiv.1601.01229) |

## Submission history

From: Guido Schmitz \[[view email](https://arxiv.org/show-email/f0a5802e/1601.01229)\]  
**[\[v1\]](https://arxiv.org/abs/1601.01229v1)** Wed, 6 Jan 2016 16:20:33 UTC (88 KB)  
**[\[v2\]](https://arxiv.org/abs/1601.01229v2)** Thu, 7 Jan 2016 09:09:59 UTC (88 KB)  
**[\[v3\]](https://arxiv.org/abs/1601.01229v3)** Fri, 27 May 2016 09:37:26 UTC (112 KB)  
**\[v4\]** Mon, 8 Aug 2016 15:42:17 UTC (111 KB)

[Which authors of this paper are endorsers?](https://arxiv.org/auth/show-endorsers/1601.01229) | Disable MathJax ([What is MathJax?](https://info.arxiv.org/help/mathjax.html))