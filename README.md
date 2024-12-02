# block-stats

This project indexes OPCODE stats by block by transaction (and potential other stats)

#### build

```shell
make build
```

#### run

This will walk backwards from latest block, and create a data file of each transaction's OPCODE counts. 

```shell
 target/opcode-stats run --url https://rpc-debug-erigon.zkevm-g-mainnet.com/ --output data.json --overwrite --concurrency 10
```

TODO: write a reducer program to analyze the data file 



