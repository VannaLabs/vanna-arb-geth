package inference

import (
	"context"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	engine "github.com/ethereum/go-ethereum/engine"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

/**
Proto Package Installation:
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

Protoc generation:
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    inference.proto
*/

type EngineNode struct {
	PublicKey  string
	IPAddress  string
	EthAddress string
	Stake      float32
}

type InferenceTx struct {
	Hash     string
	Seed     string
	Pipeline string
	Model    string
	Params   string
	TxType   string
}

type InferenceConsolidation struct {
	Tx           InferenceTx
	Result       string
	Attestations []string
	Weight       float32
}

type InferenceConsensus struct {
	resultMap map[string]InferenceConsolidation
	mu        sync.Mutex
}

func (ic InferenceConsolidation) attest(threshold float32, node EngineNode, result InferenceResult, nodeWeight float32) bool {
	if !node.validateInference(result) {
		return ic.Weight >= threshold
	}
	ic.Attestations = append(ic.Attestations, node.PublicKey)
	ic.Weight += nodeWeight
	return ic.Weight >= threshold
}

func (engineNode EngineNode) validateInference(result InferenceResult) bool {
	return true
}

type RequestClient struct {
	port  int
	txMap map[string]float64
}

// Instantiating a new request client
func NewRequestClient(portNum int) *RequestClient {
	rc := &RequestClient{
		port: portNum,
	}
	return rc
}

// Emit inference transaction
func (rc RequestClient) Emit(tx InferenceTx) (string, error) {
	consensus := InferenceConsensus{resultMap: make(map[string]InferenceConsolidation)}
	resultChan := make(chan string)
	errorChan := make(chan bool)
	nodes := getNodes()
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go rc.emitToNode(consensus, node, tx, resultChan, errorChan)
	}

	timestamp := time.Now().Unix()
	go func() {
		timeout := transactionTimeout(tx)
		for time.Now().Unix()-timestamp < timeout {
			time.Sleep(3 * time.Second)
		}
		errorChan <- true
		wg.Wait()
		close(resultChan)
	}()

	select {
	case output := <-resultChan:
		triggerEvaluate(consensus)
		return output, nil
	case <-errorChan:
		triggerEvaluate(consensus)
		return "INFERENCE ERROR", errors.New("Could not reach consensus")
	}

}

func (rc RequestClient) emitToNode(consensus InferenceConsensus, node EngineNode, tx InferenceTx, resultChan chan<- string, errorChan chan<- bool) {
	serverAddr := getAddress(node.IPAddress, rc.port)
	opts := getDialOptions()
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		return
	}
	defer conn.Close()
	client := NewInferenceClient(conn)
	var result InferenceResult
	var inferErr error
	if tx.TxType == "inference" {
		result, inferErr = RunInference(client, tx)
	} else if tx.TxType == "pipeline" {
		result, inferErr = RunPipeline(client, tx)
	}
	if inferErr != nil {
		errorChan <- true
		//log.Fatalf("RPC Inference Failed: %v", err)
		return
	}
	valid, err := validateSignature(node, result)
	if err != nil || !valid {
		return
	}
	consensus.mu.Lock()
	if _, ok := consensus.resultMap[result.Value]; !ok {
		consensus.resultMap[result.Value] = InferenceConsolidation{Tx: tx, Result: result.Value, Attestations: []string{}, Weight: 0}
	}
	// Increment results count
	if val, ok := consensus.resultMap[result.Value]; ok {
		complete := val.attest(engine.GetWeightThreshold(), node, result, node.Stake)
		if complete {
			resultChan <- val.Result
		}
	}
	consensus.mu.Unlock()
	return
}

// Runs inference request via gRPC
func RunInference(client InferenceClient, tx InferenceTx) (InferenceResult, error) {
	inferenceParams := buildInferenceParameters(tx)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := client.RunInference(ctx, inferenceParams)
	if err != nil {
		return InferenceResult{}, errors.New("Inference Execution Failed")
	}
	return *result, nil
}

// Runs pipeline  request via gRPC
func RunPipeline(client InferenceClient, tx InferenceTx) (InferenceResult, error) {
	pipelineParams := buildPipelineParameters(tx)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := client.RunPipeline(ctx, pipelineParams)
	if err != nil {
		return InferenceResult{}, errors.New("Inference Execution Failed")
	}
	return *result, nil
}

// Get IP addresses of inference nodes on network
func getNodes() []EngineNode {
	nodeInfo := engine.NodeLookup()
	nodes := []EngineNode{}
	for i := 0; i < len(nodeInfo); i++ {
		nodes = append(nodes,
			EngineNode{
				PublicKey:  nodeInfo[i].PublicKey,
				IPAddress:  nodeInfo[i].IP,
				EthAddress: nodeInfo[i].Address,
				Stake:      nodeInfo[i].Stake,
			})
	}
	return nodes
}

func getAddress(ip string, port int) string {
	return ip + ":" + strconv.Itoa(port)
}

func getDialOptions() []grpc.DialOption {
	var opts []grpc.DialOption
	// TODO: Add TLS and security auth measures
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	return opts
}

func buildInferenceParameters(tx InferenceTx) *InferenceParameters {
	return &InferenceParameters{Tx: tx.Hash, ModelHash: tx.Model, ModelInput: tx.Params}
}

func buildPipelineParameters(tx InferenceTx) *PipelineParameters {
	return &PipelineParameters{
		Tx:           tx.Hash,
		Seed:         tx.Seed,
		PipelineName: tx.Pipeline,
		ModelHash:    tx.Model,
		ModelInput:   tx.Params,
	}
}

func validateSignature(engineNode EngineNode, result InferenceResult) (bool, error) {
	return true, nil
}

func HexToBytes(hexString string) ([]byte, error) {
	// Remove any "0x" prefix if present
	if len(hexString) >= 2 && hexString[:2] == "0x" {
		hexString = hexString[2:]
	}

	// Check if the hex string has an odd length (invalid)
	if len(hexString)%2 != 0 {
		return nil, errors.New("Hex string has odd length")
	}

	// Decode the hex string to bytes
	bytes, err := hex.DecodeString(hexString)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// Evaluates node behavior
func triggerEvaluate(ic InferenceConsensus) {
	return
}

func transactionTimeout(tx InferenceTx) int64 {
	return 3
}
