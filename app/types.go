package app

type Payload struct {
	Block    uint64         `json:"block"`
	Tx       int            `json:"tx"`
	Hash     string         `json:"hash"`
	Contract string         `json:"contract"`
	Data     map[string]int `json:"data"`
}
