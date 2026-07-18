## Installation

There are three ways to run Apalache:

1. [Prebuilt package](https://apalache-mc.org/docs/apalache/installation/jvm.html): download a prebuilt package and run it in the JVM.
2. [Docker](https://apalache-mc.org/docs/apalache/installation/docker.html): download and run a Docker image.
3. [Build from source](https://apalache-mc.org/docs/apalache/installation/source.html): build Apalache from sources and run the compiled package.

If you just want to try the tool, we recommend using the [prebuilt package](https://apalache-mc.org/docs/apalache/installation/jvm.html).

## System requirements

**Memory**: Apalache uses a backend SMT solver, Microsoft Z3 by default, and the required memory largely depends on the selected solver and specification. We recommend to allocate at least 4GB of memory for the tool.