## gofail

[![Build Status](https://camo.githubusercontent.com/adadfa2546c7f196344caaafde6d868e0357d7183f32593cd6e60ec9c9dec55e/68747470733a2f2f7472617669732d63692e636f6d2f657463642d696f2f676f6661696c2e7376673f6272616e63683d6d6173746572)](https://travis-ci.com/etcd-io/gofail)

An implementation of [failpoints](http://www.freebsd.org/cgi/man.cgi?query=fail) for golang. Please read [design.md](https://github.com/etcd-io/gofail/blob/master/doc/design.md) for a deeper understanding.

## Add a failpoint

Failpoints are special comments that include a failpoint variable declaration and some trigger code,

```
func someFunc() string {
    // gofail: var SomeFuncString string
    // // this is called when the failpoint is triggered
    // return SomeFuncString
    return "default"
}
```

## Build with failpoints

Building with failpoints will translate gofail comments in place to code that accesses the gofail runtime.

Call gofail in the directory with failpoints to generate gofail runtime bindings, then build as usual,

```
gofail enable
go build cmd/
```

The translated code looks something like,

```
func someFunc() string {
    if vSomeFuncString, __fpErr := __fp_SomeFuncString.Acquire(); __fpErr == nil { SomeFuncString, __fpTypeOK := vSomeFuncString.(string); if !__fpTypeOK { goto __badTypeSomeFuncString} 
         // this is called when the failpoint is triggered
         return SomeFuncString; goto __nomockSomeFuncString; __badTypeSomeFuncString: __fp_SomeFuncString.BadType(vSomeFuncString, "string"); __nomockSomeFuncString: };
    return "default"
}
```

To disable failpoints and revert to the original code,

```
gofail disable
```

## Triggering a failpoint

After building with failpoints enabled, the program's failpoints can be activated so they may trigger when evaluated.

### Command line

From the command line, trigger the failpoint to set SomeFuncString to `hello`,

```
GOFAIL_FAILPOINTS='SomeFuncString=return("hello")' ./cmd
```

Multiple failpoints are set by using ';' for a delimiter,

```
GOFAIL_FAILPOINTS='failpoint1=return("hello");failpoint2=sleep(10)' ./cmd
```

### HTTP endpoint

First, enable the HTTP server from the command line,

```
GOFAIL_HTTP="127.0.0.1:1234" ./cmd
```

Activate a failpoint with curl,

```
$ curl http://127.0.0.1:1234/SomeFuncString -XPUT -d'return("hello")'
```

List the failpoints,

```
$ curl http://127.0.0.1:1234/SomeFuncString=return("hello")
```

Retrieve the execution count of a failpoint,

```
$curl http://127.0.0.1:1234/SomeFuncString/count -XGET
```

Deactivate a failpoint,

```
$ curl http://127.0.0.1:1234/SomeFuncString -XDELETE
```

### Unit tests

From a unit test,

```
import (
    "testing"

    gofail "go.etcd.io/gofail/runtime"
)

func TestWhatever(t *testing.T) {
    gofail.Enable("SomeFuncString", \`return("hello")\`)
    defer gofail.Disable("SomeFuncString")
    ...
}
```