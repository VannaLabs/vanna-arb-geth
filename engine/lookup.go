package engine

import "errors"

type NodeInfo struct {
	IP        string
	PublicKey string
	Address   string
	Stake     float32
}

type NodeBalancer struct {
	nodeMap map[string]NodeInfo
}

func GetWeightThreshold() float32 {
	return 0.1
}

func GetNodeBalancer() NodeBalancer {
	var nodeBalancer NodeBalancer
	nodeBalancer.nodeMap = make(map[string]NodeInfo)
	nodeBalancer.nodeMap["3.135.202.160"] =
		NodeInfo{
			IP:        "3.135.202.160",
			PublicKey: "046fcc37ea5e9e09fec6c83e5fbd7a745e3eee81d16ebd861c9e66f55518c197984e9f113c07f875691df8afc1029496fc4cb9509b39dcd38f251a83359cc8b4f7",
			Address:   "0x123",
			Stake:     0.1,
		}
	return nodeBalancer
}

func (balancer NodeBalancer) NodeLookup() ([]NodeInfo, error) {
	// TODO: Do formal lookup and load balancing logic
	return []NodeInfo{balancer.nodeMap["3.135.202.160"]}, nil
}

func (balancer NodeBalancer) PrivateNodeLookup(ip string) ([]NodeInfo, error) {
	node, ok := balancer.nodeMap[ip]
	if ok {
		return []NodeInfo{node}, nil
	} else {
		return []NodeInfo{}, errors.New("Could not find node " + ip)
	}
}
