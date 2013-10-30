## About

This is a memcache client library for the Go programming language
(http://golang.org/). This is a high performance fork of the original
library at http://github.com/bradfitz/gomemcache. The following is
a comparison between the original library and this one:

    benchmark                               old ns/op    new ns/op    delta
    BenchmarkSetGet                            214443       175154  -18.32%
    BenchmarkSetGetLarge                       262164       196155  -25.18%
    BenchmarkConcurrentSetGetSmall10_100     82561221     62172865  -24.69%
    BenchmarkConcurrentSetGetLarge10_100     96067285     74113235  -22.85%
    BenchmarkConcurrentSetGetSmall20_100    152834658    116654143  -23.67%
    BenchmarkConcurrentSetGetLarge20_100    202574186    144950678  -28.45%

    benchmark                                old MB/s     new MB/s  speedup
    BenchmarkSetGet                              0.03         0.03    1.00x
    BenchmarkSetGetLarge                         4.82         6.44    1.34x
    BenchmarkConcurrentSetGetSmall10_100         0.07         0.10    1.43x
    BenchmarkConcurrentSetGetLarge10_100        13.16        17.05    1.30x
    BenchmarkConcurrentSetGetSmall20_100         0.08         0.10    1.25x
    BenchmarkConcurrentSetGetLarge20_100        12.48        17.44    1.40x

    benchmark                              old allocs   new allocs    delta
    BenchmarkSetGet                                18            6  -66.67%
    BenchmarkSetGetLarge                           19            6  -68.42%
    BenchmarkConcurrentSetGetSmall10_100        58469         6268  -89.28%
    BenchmarkConcurrentSetGetLarge10_100        59848         6277  -89.51%
    BenchmarkConcurrentSetGetSmall20_100       117177        12663  -89.19%
    BenchmarkConcurrentSetGetLarge20_100       120173        12686  -89.44%

    benchmark                               old bytes    new bytes    delta
    BenchmarkSetGet                              2479          170  -93.14%
    BenchmarkSetGetLarge                         7537         1184  -84.29%
    BenchmarkConcurrentSetGetSmall10_100      3101520       203867  -93.43%
    BenchmarkConcurrentSetGetLarge10_100      8330341      1211143  -85.46%
    BenchmarkConcurrentSetGetSmall20_100      6318072       421952  -93.32%
    BenchmarkConcurrentSetGetLarge20_100     16884200      2437906  -85.56%

## Installing

### Using *go get*

    $ go get github.com/rainycape/gomemcache/memcache

After this command *gomemcache* is ready to use. Its source will be in:

    $GOROOT/src/pkg/github.com/rainycape/gomemcache/memcache

You can use `go get -u -a` for update all installed packages.

### Using *git clone* command:

    $ git clone git://github.com/rainycape/gomemcache
    $ cd gomemcache/memcache
    $ make install

## Example

    import (
            "github.com/rainycape/gomemcache/memcache"
    )

    func main() {
         mc := memcache.New("10.0.0.1:11211", "10.0.0.2:11211", "10.0.0.3:11212")
         mc.Set(&memcache.Item{Key: "foo", Value: []byte("my value")})

         it, err := mc.Get("foo")
         ...
    }

## Full docs, see:

See http://gopkgdoc.appspot.com/pkg/github.com/rainycape/gomemcache/memcache

Or run:

    $ godoc github.com/rainycape/gomemcache/memcache

