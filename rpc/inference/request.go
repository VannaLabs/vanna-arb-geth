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

const (
	Inference   = "inference"
	ZKInference = "zkinference"
	OPInference = "opinference"
)

type InferenceNode struct {
	PublicKey  string
	IPAddress  string
	EthAddress string
	Stake      float32
}

type InferenceTx struct {
	Hash      string
	Seed      string
	Pipeline  string
	Model     string
	Params    string
	TxType    string
	Value     string
	ZKPayload ZKP
}

type ZKP struct {
	proof    string
	settings string
	vk       []byte
	srs      string
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

func (ic InferenceConsolidation) attest(threshold float32, node InferenceNode, result InferenceResult, nodeWeight float32) bool {
	if !node.inferenceCheck(result) {
		return ic.Weight >= threshold
	}
	ic.Attestations = append(ic.Attestations, node.PublicKey)
	ic.Weight += nodeWeight
	return ic.Weight >= threshold
}

func (engineNode InferenceNode) inferenceCheck(result InferenceResult) bool {
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
	nodes := getNodes(tx.TxType)
	consensus := InferenceConsensus{resultMap: make(map[string]InferenceConsolidation)}
	resultChan := make(chan string)
	errorChan := make(chan string)
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go rc.emitToNode(consensus, node, tx, resultChan, errorChan)
	}

	timestamp := time.Now().Unix()
	go func() {
		timeout := transactionTimeout(tx)
		for time.Now().Unix() < (timestamp + transactionTimeout(tx)) {
			time.Sleep(time.Duration(1))
		}
		errorChan <- "Timeout exceeded " + strconv.FormatInt(timeout, 10) + " seconds"
		wg.Wait()
		close(resultChan)
	}()

	select {
	case output := <-resultChan:
		triggerEvaluate(tx, consensus)
		return output, nil
	case <-errorChan:
		triggerEvaluate(tx, consensus)
		return "INFERENCE ERROR", errors.New("Could not reach inference consensus")
	}
}

func (rc RequestClient) emitToNode(consensus InferenceConsensus, node InferenceNode, tx InferenceTx, resultChan chan<- string, errorChan chan<- string) {
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
	if tx.TxType == Inference {
		result, inferErr = RunInference(client, tx)
	} else if tx.TxType == ZKInference {
		var zkresult ZKInferenceResult
		zkresult, inferErr = RunZKInference(client, tx)
		if !validateZKProof(zkresult) {
			errorChan <- "ZKML Proof cannot be validated"
		}
		result = InferenceResult{Tx: zkresult.Tx, Node: zkresult.Node, Value: zkresult.Value}
	} else if tx.TxType == "pipeline" {
		result, inferErr = RunPipeline(client, tx)
	}
	if inferErr != nil {
		errorChan <- inferErr.Error()
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
		return InferenceResult{}, errors.New(err.Error())
	}
	return *result, nil
}

// Runs zkml-secured inference request via gRPC
func RunZKInference(client InferenceClient, tx InferenceTx) (ZKInferenceResult, error) {
	inferenceParams := buildInferenceParameters(tx)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(transactionTimeout(tx))*time.Second)
	defer cancel()
	result, err := client.RunZKInference(ctx, inferenceParams)
	if err != nil {
		return ZKInferenceResult{}, errors.New(err.Error())
	}
	return *result, nil
}

// Runs pipeline  request via gRPC
func RunPipeline(client InferenceClient, tx InferenceTx) (InferenceResult, error) {
	pipelineParams := buildPipelineParameters(tx)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(60*time.Second))
	defer cancel()
	result, err := client.RunPipeline(ctx, pipelineParams)
	if err != nil {
		return InferenceResult{}, errors.New("Inference Execution Failed")
	}
	return *result, nil
}

// Get IP addresses of inference nodes on network
func getNodes(txType string) []InferenceNode {
	nodeInfo := engine.NodeLookup()
	nodes := []InferenceNode{}
	for i := 0; i < len(nodeInfo); i++ {
		nodes = append(nodes,
			InferenceNode{
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

func validateSignature(engineNode InferenceNode, result InferenceResult) (bool, error) {
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

// Evaluates inference node behavior
func triggerEvaluate(tx InferenceTx, ic InferenceConsensus) {
	return
}

func transactionTimeout(tx InferenceTx) int64 {
	switch tx.TxType {
	case Inference:
		return 10
	case OPInference:
		return 10
	case ZKInference:
		return 90
	default:
		return 10
	}
}
