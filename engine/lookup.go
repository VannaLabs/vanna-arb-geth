package engine

type NodeInfo struct {
	IP        string
	PublicKey string
	Address   string
	Stake     float32
}

func GetWeightThreshold() float32 {
	return 0.1
}

func NodeLookup() []NodeInfo {
	// TODO: Do formal lookup of PoS Engine Nodes
	n := NodeInfo{
		IP:        "3.145.100.160",
		PublicKey: "046fcc37ea5e9e09fec6c83e5fbd7a745e3eee81d16ebd861c9e66f55518c197984e9f113c07f875691df8afc1029496fc4cb9509b39dcd38f251a83359cc8b4f7",
		Address:   "0x123",
		Stake:     0.1,
	}
	return []NodeInfo{n}
}
