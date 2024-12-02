package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	syslog "log"
	"os"
	"sync"

	"github.com/christophercampbell/bridge-connector/log"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/urfave/cli/v2"
)

const appName = "opcode-stats"

var (
	rpcUrl = cli.StringFlag{
		Name:     "url",
		Aliases:  []string{"u"},
		Usage:    "RPC url",
		Required: true,
	}
	startBlock = cli.Uint64Flag{
		Name:     "start-block",
		Aliases:  []string{"s"},
		Usage:    "Start block number",
		Required: false,
	}
	outputFile = cli.PathFlag{
		Name:     "output",
		Aliases:  []string{"o"},
		Usage:    "Output file for data",
		Required: true,
	}
	overwriteFile = cli.BoolFlag{
		Name:     "overwrite",
		Aliases:  []string{"w"},
		Usage:    "Overwrite output if exists",
		Required: false,
	}
	concurrencyFlag = cli.IntFlag{
		Name:     "concurrency",
		Aliases:  []string{"c"},
		Usage:    "Concurrent requests",
		Required: false,
	}
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Commands = []*cli.Command{
		{
			Name:    "run",
			Aliases: []string{},
			Usage:   fmt.Sprintf("Run the %v", appName),
			Action:  run,
			Flags:   []cli.Flag{&rpcUrl, &startBlock, &outputFile, &overwriteFile, &concurrencyFlag},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		syslog.Fatal(err)
		os.Exit(1)
	}
}

type Payload struct {
	Block    uint64         `json:"block"`
	Tx       int            `json:"tx"`
	Hash     string         `json:"hash"`
	Contract string         `json:"contract"`
	Data     map[string]int `json:"data"`
}

func run(cliCtx *cli.Context) error {

	output := cliCtx.Path(outputFile.Name)
	overwrite := cliCtx.Bool(overwriteFile.Name)
	concurrency := cliCtx.Int(concurrencyFlag.Name)
	if concurrency == 0 {
		concurrency = 1
	}

	messages := make(chan Payload)

	log.Init("info", "stderr")
	log.Infof("starting data collector to '%s' with concurrency = %d", output, concurrency)

	go writeMessages(output, overwrite, messages)

	if err := log.Init("info", "stderr"); err != nil {
		return err
	}

	client, err := jsonrpc.NewClient(cliCtx.String(rpcUrl.Name))
	if err != nil {
		return err
	}
	defer client.Close()

	latest, err := client.Eth().BlockNumber()
	if err != nil {
		return err
	}

	startAt := latest
	if cliCtx.IsSet(startBlock.Name) {
		startAt = cliCtx.Uint64(startBlock.Name)
	}

	// Walks backward from start to 0, N at a time
	for i := startAt; i > 0; i-- {
		var wg sync.WaitGroup
		for w := 0; w < 5; w++ {
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
		return
	}

	for i := 0; i < len(block.Transactions); i++ {
		tx := block.Transactions[i]
		txHash := tx.Hash
		if tx.To == nil {
			continue
		}
		var trace *jsonrpc.TransactionTrace
		trace, err = client.Debug().TraceTransaction(txHash) // might have to customize this to use particular analyzer
		if err != nil {
			log.Error(err)
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
			panic(err)
		}
	} else if _, err = os.Stat(path); err == nil {
		panic(fmt.Sprintf("output file '%s' already exists, choose another", path))
	} else if file, err = os.Open(path); err != nil {
		panic(err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for {
		payload := <-messages
		var msg []byte
		msg, err = json.Marshal(payload)
		if err != nil {
			continue
		}
		writer.WriteString(string(msg))
		writer.WriteString("\n")
		_ = writer.Flush()
	}
}
