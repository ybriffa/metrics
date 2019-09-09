# metrics
metrics is a library to asbtract all the management of the metrics initialization, registry management and metrics push. it relies on github.com/rcrowley/go-metrics

Different types of senders are availables :
- logrus
- warp10
- http

# usage
In order to use in your code, you basically just have to :

```go
package main

import (
       "github.com/ybriffa/metrics"
)

func init() {
     err := metrics.Init("application-name")
     if err != nil {
        // maybe do something
     }
}

func main() {
     //...

     metrics.Register(registry-name, registry, tags)
     //Or
     metrics.RegisterStruct(registry-name, &struct-registry, tags)
     
}

```

Once the regitry is registered, it will be automatically managed by the lib, and will be pushed without doing anything.

The Register function takes a github.com/rcrowley/go-metrics.Registry, so it has to be declared and fully instanciated previously.

The RegisterStruct function takes a pointer to a struct containing different kind of github.com/rcrowley/go-metrics (such as meter, counter, gauge...), creates its own registry, instanciate all the metrics for both its registry and the structure given, and finally uses Regiter.

You have to silent import the drivers you want to use.