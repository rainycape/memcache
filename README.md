## About

This is a memcache client library for the Go programming language
(http://golang.org/). This is high performance fork of the original
library at http://github.com/bradfitz/gomemcache. The following is
a comparison between the original library and this one:

    benchmark                               old ns/op    new ns/op    delta
    BenchmarkSetGet                            214443       192399  -10.28%
    BenchmarkSetGetLarge                       262164       200473  -23.53%
    BenchmarkConcurrentSetGetSmall10_100     82561221     70219863  -14.95%
    BenchmarkConcurrentSetGetLarge10_100     96067285     79718583  -17.02%
    BenchmarkConcurrentSetGetSmall20_100    152834658    127891303  -16.32%
    BenchmarkConcurrentSetGetLarge20_100    202574186    153507440  -24.22%
    
    benchmark                                old MB/s     new MB/s  speedup
    BenchmarkSetGet                              0.03         0.03    1.00x
    BenchmarkSetGetLarge                         4.82         6.31    1.31x
    BenchmarkConcurrentSetGetSmall10_100         0.07         0.09    1.29x
    BenchmarkConcurrentSetGetLarge10_100        13.16        15.86    1.21x
    BenchmarkConcurrentSetGetSmall20_100         0.08         0.09    1.12x
    BenchmarkConcurrentSetGetLarge20_100        12.48        16.47    1.32x
    
    benchmark                              old allocs   new allocs    delta
    BenchmarkSetGet                                18           10  -44.44%
    BenchmarkSetGetLarge                           19           10  -47.37%
    BenchmarkConcurrentSetGetSmall10_100        58469        10276  -82.42%
    BenchmarkConcurrentSetGetLarge10_100        59848        10263  -82.85%
    BenchmarkConcurrentSetGetSmall20_100       117177        20705  -82.33%
    BenchmarkConcurrentSetGetLarge20_100       120173        20650  -82.82%
    
    benchmark                               old bytes    new bytes    delta
    BenchmarkSetGet                              2479          202  -91.85%
    BenchmarkSetGetLarge                         7537         1217  -83.85%
    BenchmarkConcurrentSetGetSmall10_100      3101520       235682  -92.40%
    BenchmarkConcurrentSetGetLarge10_100      8330341      1239295  -85.12%
    BenchmarkConcurrentSetGetSmall20_100      6318072       479380  -92.41%
    BenchmarkConcurrentSetGetLarge20_100     16884200      2501016  -85.19%


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

