package app

import (
	"bufio"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

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

	blocks := make(chan uint64, 1)

	go func(start uint64, blockNums chan uint64) {
		for start > 0 {
			blockNums <- start
			start--
		}
	}(startAt, blocks)

	for i := 0; i < concurrency; i++ {
		go traceTxs(client, blocks, messages)
	}

	BlockOnInterrupts()

	return nil
}

func traceTxs(client *jsonrpc.Client, blockNums chan uint64, messages chan Payload) {
	for {
		blockNum := <-blockNums
		if blockNum <= 0 {
			return
		}

		log.Info("block: ", blockNum)

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

// DefaultInterruptSignals is a set of default interrupt signals.
var DefaultInterruptSignals = []os.Signal{
	os.Interrupt,
	os.Kill,
	syscall.SIGTERM,
	syscall.SIGQUIT,
}

// BlockOnInterrupts blocks until a SIGTERM is received.
// Passing in signals will override the default signals.
func BlockOnInterrupts(signals ...os.Signal) {
	if len(signals) == 0 {
		signals = DefaultInterruptSignals
	}
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, signals...)
	<-interruptChannel
}
