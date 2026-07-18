## About the Conformance Suite

The conformance suite is an open source project run by the OpenID Foundation, the source code can be found on gitlab: [https://gitlab.com/openid/conformance-suite/](https://gitlab.com/openid/conformance-suite/). There is no cost to utilize the conformance suite to test OpenID deployments and is available for all to utilize at any time. A fee is required for OpenID certifications.

Instructions for running the suite are found on: [http://openid.net/certification/instructions/](http://openid.net/certification/instructions/)

Release notes for the suite are available on gitlab: [https://gitlab.com/openid/conformance-suite/-/releases](https://gitlab.com/openid/conformance-suite/-/releases)

If you need access to bug fixes that have been merged into the conformance suite but are not yet pushed to the production server, you can use the staging environment (which automatically reflect the ‘master’ branch in git): [https://staging.certification.openid.net/](https://staging.certification.openid.net/) (this would require you to add redirect urls for the staging server to your clients).

The master git branch is regression tested against various vendor-provided cloud environments at least once every 24 hours; a summary of the results can be viewed here: [https://staging.certification.openid.net/plans.html?public=true](https://staging.certification.openid.net/plans.html?public=true)

The conformance suite can be installed locally inside docker, see: [https://gitlab.com/openid/conformance-suite/wikis/Developers/Build-&-Run](https://gitlab.com/openid/conformance-suite/wikis/Developers/Build-&-Run)

A python script and library are available to allow the conformance suite to be used in a continuous integration system; it is highly recommended that authorization server developers integrate this into their development pipeline: [https://gitlab.com/openid/conformance-suite/blob/master/scripts/run-test-plan.py](https://gitlab.com/openid/conformance-suite/blob/master/scripts/run-test-plan.py)

Enhancements, bug fixes and other contributions to the conformance suite are welcome, see: [https://gitlab.com/openid/conformance-suite/wikis/Developers/Contributing](https://gitlab.com/openid/conformance-suite/wikis/Developers/Contributing)

Please report bugs in the issue tracker with FULL details of what happened, links to the log page for the test run, and so on: [https://gitlab.com/openid/conformance-suite/issues](https://gitlab.com/openid/conformance-suite/issues)

If you require support, please email [certification@oidf.org](mailto:certification@oidf.org).

![](https://openid.net/wp-content/uploads/2022/11/dots-getcertified-2048x1301.png)