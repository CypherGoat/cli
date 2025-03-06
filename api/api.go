package api

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const URL = "api.cyphergoat.com"

//go:embed .env
var envFile embed.FS
var API_KEY string

type Estimate struct {
	ExchangeName  string  `json:"Exchange"`
	ReceiveAmount float64 `json:"Amount"`
	MinAmount     float64 `json:"MinAmount"`
	Network1      string
	Network2      string
	Coin1         string
	Coin2         string
	SendAmount    float64
	Address       string
	ImageURL      string
}

type TransactionResponse struct {
	Transaction Transaction `json:"transaction"`
}
type Transaction struct {
	Coin1          string    `json:"Coin1,omitempty"`
	Coin2          string    `json:"Coin2,omitempty"`
	Network1       string    `json:"Network1,omitempty"`
	Network2       string    `json:"Network2,omitempty"`
	Address        string    `json:"Address,omitempty"`
	EstimateAmount float64   `json:"EstimateAmount,omitempty"`
	Provider       string    `json:"Provider,omitempty"`
	Id             string    `json:"Id,omitempty"`
	SendAmount     float64   `json:"SendAmount,omitempty"`
	Track          string    `json:"Track,omitempty"`
	Status         string    `json:"Status,omitempty"`
	KYC            string    `json:"KYC,omitempty"`
	Token          string    `json:"Token,omitempty"`
	Done           bool      `json:"Done,omitempty"`
	CGID           string    `json:"CGID,omitempty"`
	CreatedAt      time.Time `json:"CreatedAt,omitempty"`
}

func init() {
	envData, err := envFile.ReadFile(".env")
	if err != nil {
		fmt.Println("Error reading embedded .env file")
		return
	}

	envMap, err := godotenv.Unmarshal(string(envData))
	if err != nil {
		fmt.Println("Error unmarshalling .env file")
		return
	}

	for key, value := range envMap {
		os.Setenv(key, value)
	}

	API_KEY = os.Getenv("API_KEY")
}

func SendRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+API_KEY)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var responseMap map[string]interface{}
	err = json.Unmarshal(data, &responseMap)
	if err != nil {
		return nil, err
	}

	if errStr, ok := responseMap["error"].(string); ok {
		return nil, fmt.Errorf("%s", errStr)
	}

	return data, nil
}

func FetchEstimateFromAPI(coin1, coin2 string, amount float64, best bool, network1, network2 string) ([]Estimate, error) {
	var url string
	if best {
		url = fmt.Sprintf("https://%s/estimate?coin1=%s&coin2=%s&amount=%f&network1=%s&network2=%s&best=true", URL, coin1, coin2, amount, network1, network2)
	} else {
		url = fmt.Sprintf("https://%s/estimate?coin1=%s&coin2=%s&amount=%f&network1=%s&network2=%s&best=false", URL, coin1, coin2, amount, network1, network2)
	}

	data, err := SendRequest(url)
	if err != nil {
		return nil, err
	}

	var responseMap map[string]interface{}
	err = json.Unmarshal(data, &responseMap)
	if err != nil {
		return nil, err
	}

	if errStr, ok := responseMap["error"].(string); ok {
		return nil, fmt.Errorf("%s", errStr)
	}

	type ApiResponse struct {
		Rates []Estimate `json:"rates"`
	}

	var result ApiResponse
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	sort.Slice(result.Rates, func(i, j int) bool {
		return result.Rates[i].ReceiveAmount > result.Rates[j].ReceiveAmount
	})

	for i := range result.Rates {
		result.Rates[i].Coin1 = coin1
		result.Rates[i].Coin2 = coin2
		result.Rates[i].SendAmount = amount
		result.Rates[i].Network1 = network1
		result.Rates[i].Network2 = network2
	}

	return result.Rates, nil
}

func CreateTradeFromAPI(coin1, coin2 string, amount float64, address, partner string, network1, network2 string) (error, Transaction) {
	url := fmt.Sprintf("https://%s/swap?coin1=%s&coin2=%s&amount=%f&partner=%s&address=%s&network1=%s&network2=%s", URL, coin1, coin2, amount, partner, address, network1, network2)

	data, err := SendRequest(url)
	if err != nil {
		return err, Transaction{}
	}

	var result TransactionResponse

	err = json.Unmarshal([]byte(data), &result)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err, Transaction{}
	}

	transaction := result.Transaction
	fmt.Printf("Transaction: %+v\n", transaction)
	return nil, transaction
}

func TrackTxFromAPI(t Transaction) (error, Transaction) {
	url := fmt.Sprintf("https://%s/transaction?id=%s", URL, strings.ToLower(t.Provider), t.Id)
	data, err := SendRequest(url)
	if err != nil {
		return err, t
	}

	var responseMap map[string]interface{}
	err = json.Unmarshal(data, &responseMap)
	if err != nil {
		return err, t
	}

	status, ok := responseMap["status"].(string)
	if !ok {
		return fmt.Errorf("status field is missing or not a string"), Transaction{}

	}
	t.Status = status

	return nil, t

}

func GetTransactionFromAPI(id string) (error, Transaction) {
	url := fmt.Sprintf("https://%s/transaction?id=%s", URL, id)
	data, err := SendRequest(url)
	if err != nil {
		return err, Transaction{}
	}

	var result map[string]Transaction
	err = json.Unmarshal(data, &result)
	if err != nil {
		return err, Transaction{}
	}

	transaction := result["transaction"]
	return nil, transaction
}
