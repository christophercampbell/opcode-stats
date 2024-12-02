package app

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"

	"github.com/christophercampbell/bridge-connector/log"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/urfave/cli/v2"
)

func Run(cliCtx *cli.Context) error {

	output := cliCtx.Path(outputFlag.Name)
	overwrite := cliCtx.Bool(overwriteFlag.Name)
	concurrency := cliCtx.Int(concurrencyFlag.Name)
	if concurrency == 0 {
		concurrency = 1
	}

	messages := make(chan Payload)

	log.Init("info", "stderr")
	log.Infof("starting data collector to '%s' with concurrency = %d", output, concurrency)

	go writeMessages(output, overwrite, messages)

	client, err := jsonrpc.NewClient(cliCtx.String(rpcUrlFlag.Name))
	if err != nil {
		return err
	}
	defer client.Close()

	latest, err := client.Eth().BlockNumber()
	if err != nil {
		return err
	}

	startAt := latest
	if cliCtx.IsSet(startBlockFlag.Name) {
		startAt = cliCtx.Uint64(startBlockFlag.Name)
	}

	// Walks backward from start to 0, N at a time
	for i := startAt; i > 0; i-- {
		var wg sync.WaitGroup
		for w := 0; w < concurrency; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				traceTxs(client, i, messages)
			}()
		}
		wg.Wait()
	}

	return nil
}

func traceTxs(client *jsonrpc.Client, blockNum uint64, messages chan Payload) {
	block, err := client.Eth().GetBlockByNumber(ethgo.BlockNumber(blockNum), true)
	if err != nil {
		log.Error(err)
		return
	}
	for i := 0; i < len(block.Transactions); i++ {
		tx := block.Transactions[i]
		txHash := tx.Hash
		if tx.To == nil {
			continue
		}
		var trace *jsonrpc.TransactionTrace
		trace, err = client.Debug().TraceTransaction(txHash)
		if err != nil {
			log.Error(err)
			continue
		}
		if trace == nil {
			continue
		}
		ops := make(map[string]int)
		for k := 0; k < len(trace.StructLogs); k++ {
			log := trace.StructLogs[k]
			if count, ok := ops[log.Op]; ok {
				ops[log.Op] = count + 1
			} else {
				ops[log.Op] = 1
			}
		}

		payload := Payload{
			Block:    blockNum,
			Tx:       i,
			Hash:     tx.Hash.String(),
			Contract: tx.To.String(),
			Data:     ops,
		}

		messages <- payload
	}
}

func writeMessages(path string, overwrite bool, messages chan Payload) {
	var (
		err  error
		file *os.File
	)
	if overwrite {
		file, err = os.Create(path)
		if err != nil {
			log.Fatalf("%v", err)
		}
	} else if _, err = os.Stat(path); err == nil {
		log.Fatalf("output file '%s' already exists, choose another", path)
	} else if file, err = os.Open(path); err != nil {
		log.Fatalf("%v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for {
		payload := <-messages
		var msg []byte
		msg, err = json.Marshal(payload)
		if err != nil {
			log.Errorf("error marshalling data: %v", err)
			continue
		}
		writer.WriteString(string(msg))
		writer.WriteString("\n")
		_ = writer.Flush()
	}
}
