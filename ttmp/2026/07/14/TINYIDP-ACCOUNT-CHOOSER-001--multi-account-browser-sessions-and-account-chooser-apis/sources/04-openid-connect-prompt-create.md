|  | OpenID Connect Prompt Create | December 2022 |
| --- | --- | --- |
| Fletcher | Standards Track | \[Page\] |

## Abstract

An extension to the OpenID Connect Authentication Framework defining a new value for the `prompt` parameter that instructs the OpenID Provider to start the user account creation experience and after the user account has been created return the requested tokens to the client to complete the authentication flow.[¶](#section-abstract-1)

## 1.

Several years of deployment and implementation experience with \[\] has uncovered a need, in some circumstances, for the client to explicitly signal to the OpenID Provider that the user desires to create a new account rather than authenticate an existing identity.[¶](#section-1-1)

This specification allows the client to indicate to the OpenID Provider that the user desires to create an account improving the user experience by reducing the friction for the user of finding the sign-up link on the default login page.[¶](#section-1-2)

This specification defines a new value for the `prompt` parameter.[¶](#section-1-3)

## 2.

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in \[\].[¶](#section-2-1)

In the.txt version of this specification, values are quoted to indicate that they are to be taken literally. When using these values in protocol messages, the quotes MUST NOT be used as part of the value. In the HTML version of this specification, values to be taken literally are indicated by the use of `this fixed-width font`.[¶](#section-2-2)

## 4.

In requests to the OpenID Provider, a client MAY indicate that the desired user experience is for the user to immediately see the user account creation UI instead of the login behavior.[¶](#section-4-1)

prompt

A value of `create` indicates to the OpenID Provider that the client desires that the user be shown the account creation UX rather than the login flow. Care must be taken if combining this value with other `prompt` values. Mutually exclusive conditions can arise so it is RECOMMENDED that `create` not be combined with any other values.[¶](#section-4-2.2)

### 4.1.

When the `prompt` parameter is used in an authorization request to the authorization endpoint with the value of `create`, it indicates that the user has chosen to be shown the account creation experience rather than the login experience. Whether the AS creates a brand new identity or helps the user authenticate an identity they already have is out of scope for this specification. The behavior is the same for all response types.[¶](#section-4.1-1)

This flow MUST NOT be considered successful without the return of a valid `id_token` or `id_token` + `access_token`. The client MUST NOT assume a new identity has been created without the successful response.[¶](#section-4.1-2)

For authorization requests sent as a JWT, such as when using Section 6 of \[\], the `prompt` claim value MUST be a space delimited set of prompt values in keeping with Section 3.1.2.1 of OpenID Connect Core 1.0.[¶](#section-4.1-3)

If the OpenID Provider receives a prompt value that it does not support (not declared in the `prompt_values_supported` metadata field) the OP SHOULD respond with an HTTP 400 (Bad Request) status code and an `error` value of `invalid_request`. It is RECOMMENDED that the OP return an `error_description` value identifying the invalid parameter value.[¶](#section-4.1-4)

In is an example of an authorization request using the `code` response type where the client tells the OpenID Provider that it wants the user to start from the account creation user experience (extra line breaks and indentation are for display purposes only).[¶](#section-4.1-5)

```
GET /as/authorization.oauth2?response_type=code
   &client_id=s6BhdRkqt3
   &state=tNwzQ87pC6llebpmac_IDeeq-mCR2wLDYljHUZUAWuI
   &redirect_uri=https%3A%2F%2Fclient%2Eexample%2Eorg%2Fcb
   &scope=openid%20profile
   &prompt=create HTTP/1.1
Host: authorization-server.example.com
```

:

### 4.2.

This specification extends the OpenID Connect Discovery Metadata \[\] and defines the following:[¶](#section-4.2-1)

prompt\_values\_supported

OPTIONAL. JSON array containing the list of prompt values that this OP supports.[¶](#section-4.2-2.2)

This metadata element is OPTIONAL in the context of the OpenID Provider not supporting the `create` value. If omitted, the Relying Party should assume that this specification is not supported. The OpenID Provider MAY provide this metadata element even if it doesn't support the `create` value.[¶](#section-4.2-3)

Specific to this specification, a value of `create` in the array indicates to the Relying party that this OpenID Provider supports this specification. If an OpenID Provider supports this specification it MUST define this metadata element in the openid-configuration file. Additionally, if this metadata element is defined by the OpenID Provider, the OP must also specify all other prompt values which it supports.[¶](#section-4.2-4)

## 5.

If integrity protecting the authorization request is required, please refer to \[\].[¶](#section-5-1)

## 6.

### 6.1.

\[OpenID.Core\]

Sakimura, N., Bradley, J., Jones, M.B., de Medeiros, B., and C. Mortimore, "OpenID Connect Core 1.0", 8 November 2014, < [http://openid.net/specs/openid-connect-core-1\_0.html](http://openid.net/specs/openid-connect-core-1_0.html) >.

\[OpenID.Discovery\]

Sakimura, N., Bradley, J., Jones, M.B., and E. Jay, "OpenID Connect Discovery 1.0", 8 November 2014, < [http://openid.net/specs/openid-connect-discovery-1\_0.html](http://openid.net/specs/openid-connect-discovery-1_0.html) >.

\[RFC2119\]

Bradner, S., "Key words for use in RFCs to Indicate Requirement Levels", BCP 14, RFC 2119, DOI 10.17487/RFC2119, March 1997, < [https://www.rfc-editor.org/info/rfc2119](https://www.rfc-editor.org/info/rfc2119) >.

\[RFC6749\]

Hardt, D., Ed., "The OAuth 2.0 Authorization Framework", RFC 6749, DOI 10.17487/RFC6749, October 2012, < [https://www.rfc-editor.org/info/rfc6749](https://www.rfc-editor.org/info/rfc6749) >.

\[RFC9101\]

Sakimura, N., Bradley, J., and M. Jones, "The OAuth 2.0 Authorization Framework: JWT-Secured Authorization Request (JAR)", RFC 9101, DOI 10.17487/RFC9101, August 2021, < [https://www.rfc-editor.org/info/rfc9101](https://www.rfc-editor.org/info/rfc9101) >.

### 6.2.

\[RFC6750\]

Jones, M. and D. Hardt, "The OAuth 2.0 Authorization Framework: Bearer Token Usage", RFC 6750, DOI 10.17487/RFC6750, October 2012, < [https://www.rfc-editor.org/info/rfc6750](https://www.rfc-editor.org/info/rfc6750) >.

## Appendix A.

This specification was developed within OpenID connect Working Group of the OpenID Foundation. Additionally, the following individuals contributed ideas, feedback, and wording that helped shape this specification:[¶](#appendix-A-1)

- Andrii Deinega [¶](#appendix-A-2.1)
- Filip Skokan [¶](#appendix-A-2.2)
- Joseph Heenan [¶](#appendix-A-2.3)
- Michael Jones [¶](#appendix-A-2.4)
- Vittorio Bertocci [¶](#appendix-A-2.5)
- William Dennis [¶](#appendix-A-2.6)

## Appendix B.

Copyright (c) 2022 The OpenID Foundation.[¶](#appendix-B-1)

The OpenID Foundation (OIDF) grants to any Contributor, developer, implementer, or other interested party a non-exclusive, royalty free, worldwide copyright license to reproduce, prepare derivative works from, distribute, perform and display, this Implementers Draft or Final Specification solely for the purposes of (i) developing specifications, and (ii) implementing Implementers Drafts and Final Specifications based on such documents, provided that attribution be made to the OIDF as the source of the material, but that such attribution does not indicate an endorsement by the OIDF.[¶](#appendix-B-2)

The technology described in this specification was made available from contributions from various sources, including members of the OpenID Foundation and others. Although the OpenID Foundation has taken steps to help ensure that the technology is available for distribution, it takes no position regarding the validity or scope of any intellectual property or other rights that might be claimed to pertain to the implementation or use of the technology described in this specification or the extent to which any license under such rights might or might not be available; neither does it represent that it has made any independent effort to identify any such rights. The OpenID Foundation and the contributors to this specification make no (and hereby expressly disclaim any) warranties (express, implied, or otherwise), including implied warranties of merchantability, non-infringement, fitness for a particular purpose, or title, related to this specification, and the entire risk as to implementing this specification is assumed by the implementer. The OpenID Intellectual Property Rights policy requires contributors to offer a patent promise not to assert certain patent claims against other contributors and against implementers. The OpenID Foundation invites any interested party to bring to its attention any copyrights, patents, patent applications, or other proprietary rights that may cover technology that may be required to practice this specification.[¶](#appendix-B-3)