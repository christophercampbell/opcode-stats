package app

import "github.com/urfave/cli/v2"

var (
	rpcUrlFlag = cli.StringFlag{
		Name:     "url",
		Aliases:  []string{"u"},
		Usage:    "RPC url",
		Required: true,
	}
	startBlockFlag = cli.Uint64Flag{
		Name:     "start-block",
		Aliases:  []string{"s"},
		Usage:    "Start block number",
		Required: false,
	}
	concurrencyFlag = cli.IntFlag{
		Name:     "concurrency",
		Aliases:  []string{"c"},
		Usage:    "Concurrent requests",
		Required: false,
	}

	RunFlags = []cli.Flag{&rpcUrlFlag, &startBlockFlag, &concurrencyFlag}
)
