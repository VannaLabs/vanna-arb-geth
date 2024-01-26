package inference

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type IrysClient struct {
	srsAccessMap map[string]int64
}

const (
	URL      = "https://gateway.irys.xyz/"
	FILEPATH = "./SRS/"
	FILETYPE = ".srs"
)

func NewIrysClient() *IrysClient {
	var ic IrysClient
	ic.srsAccessMap = make(map[string]int64)
	return &ic
}

func (ic IrysClient) getSRS(logrows string) string {
	filepath := FILEPATH + logrows + FILETYPE
	fmt.Print("File path is [" + filepath + "]")

	if _, err := os.Stat(filepath); err != nil {
		// SRS file does not exist => download from Irys
		err = ic.downloadSRS(URL+ic.getSRSMappingURL(logrows), filepath)
		fmt.Print(err)
	}
	ic.srsAccessMap[logrows] = time.Now().UnixMilli()
	return filepath
}

func (ic IrysClient) getSRSMappingURL(logrows string) string {
	return logrows
}

func (ic IrysClient) downloadSRS(url, filePath string) error {
	// Fetch content from the URL
	resp, err := http.Get(url)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Create or open the file for writing
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	fmt.Print("3")
	// Write the content to the file
	_, err = file.Write(body)
	if err != nil {
		return err
	}
	fmt.Print("4")
	return nil
}
