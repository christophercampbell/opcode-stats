# block-stats

This project indexes OPCODE stats by block by transaction (and potential other stats)

#### build

```shell
make build
```

#### run

This will walk backwards from latest block, and create a data file of each transaction's OPCODE counts. 

The program writes data to stdout, and logs to stderr

```shell
target/opcode-stats run --url https://rpc-debug-erigon.zkevm-g-mainnet.com/ --concurrency 5 > data.json
```

TODO: write a reducer program to analyze the data file 



