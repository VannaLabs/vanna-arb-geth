package inference

type zkmlClient struct {
	irysClient *IrysClient
}

func NewzkMLClient() *zkmlClient {
	var zc zkmlClient
	zc.irysClient = NewIrysClient()
	return &zc
}

func (zc zkmlClient) validateZKProof(result InferenceResult) bool {
	_ = zc.irysClient.getSRS(result.Srs)
	// TODO: Validate ezKL zkML proof
	return true
}
