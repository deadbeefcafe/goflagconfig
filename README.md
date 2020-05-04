# Yet another go configuration package

This is based off of the go flags package.  It adds support for reading and writing config files.  Config files store key/value pairs.

Config file example:
```
# this is a comment
duration = 5s
my/bool_var = true
pie = 3.14  # also a comment
my/unicode/string = 私はパイを食べたい😀
```

Usage example:
```
#!go

package main

import (
    "fmt"
    "time"

    config "github.com/deadbeefcafe/goconfig"
)

var my_bool_var bool
var my_duration time.Duration
var auger_simulate_offtime time.Duration
var pie float64

func init() {
    config.BoolVar(&my_bool_var, "my/bool_var", false, "A boolean flag that does very little")
    config.DurationVar(&my_duration, "duration", 5*time.Second, "How long to wait")
    config.Float64Var(&pie, "yum/pie", 3.14159265358979323846264, "Key lime is
    my favorite")
}

func main() {

    // config variables can also be defined in this way
    mystr := config.String("my/unicode/string", "a default str", "Just a string vaiable")
    fmt.Printf("bool = %v duration = %v pi=%f str=%s\n", my_bool_var, my_duration, pie, *mystr)

    my_bool_var = true

    config.SetFile("foo.conf")
    config.Load()
    config.Print()
    config.Save()
}

```

