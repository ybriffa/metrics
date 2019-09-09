# warp10

warp10 is a sender for to push metrics in the warp10 format

# configuration

For the code, refers to the root repository

To enable the warp10 sender you will have to set the following environement variables :

``` sh
## Mandatory
export METRICS_WARP10_ENABLE=1
export METRICS_WARP10_TOKEN="aaaabbbbbcccccc-dddd"

## Optionnal
export METRICS_WARP10_ADDR="http://anyadress/api/put" # default https://warp10.gra1-ovh.metrics.ovh.net/api/v0/update
export METRICS_WARP10_USER="my-user" # default set to the value given in the metrics.Init()
export METRICS_WARP10_PREFIX="com.ovh.my-prefix" # default set to com.ovh.engine.apiv7

```