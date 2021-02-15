package main

// get block/0x5bad55/tx/0x8784d99762bccd03b2086eabccee0d77f14d05463281e121a62abfebcf0d2d5f
import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
)

const PrintCache = "PRINTCACHE"
const GetBlockByNumber = "eth_getBlockByNumber"
const GetTxByHash = "eth_getTransactionByHash"

// this might be better to use instead of GetBlockByNumber, but the task asks to get txs them by blockNumber...
// const GetTransactionByBlockIndex = "eth_getTransactionByBlockNumberAndIndex"
const MinHashLen = 27 // an assumption that there cannot be too many indices in the block

func isHash(s string) bool {
	// works for O(1), so we can call it without worrying of efficiency
	return len(s) >= MinHashLen
}

type Printer func(...string)

type BlockStructure map[string]interface{}

func (b BlockStructure) getTransactions() []interface{} {
	return b["transactions"].([]interface{})
}

func (b BlockStructure) getNumber() string {
	return b["number"].(string)
}

type TransactionStructure map[string]interface{}

func (tx TransactionStructure) getHash() string {
	return tx["hash"].(string)
}

func (tx TransactionStructure) getTransactionIndex() string {
	return tx["transactionIndex"].(string)
}

func (tx TransactionStructure) getBlockNum() string {
	return tx["blockNumber"].(string)
}

func (tx TransactionStructure) getBlockNumTransactionIndexKey() string {
	return tx.getBlockNum() + "-" + tx.getTransactionIndex()
}

type Payload struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type Transaction struct {
	usageIndex uint
	tx         TransactionStructure
}

type ProxyCache struct {
	client       http.Client
	debugPrinter Printer
	maxTxs       uint
	cachedTxs    uint
	maxSize      uint // in bytes
	// assume that data in received JSON is approximately equals to it's string representation length
	cachedSize    uint // in bytes
	txMap         map[string] /*hash or blockHash+index*/ *Transaction
	usageIndexMap map[uint]string
	usageIndex    uint
}

func (p ProxyCache) sendRequestForTransaction(blockNum, txCode string) (TransactionStructure, error) {
	res := TransactionStructure{}
	var method string
	var params []interface{}
	callByBlockNumber := blockNum == "latest" || !isHash(txCode)
	if callByBlockNumber {
		method = GetBlockByNumber
		params = []interface{}{blockNum, true}
	} else {
		method = GetTxByHash
		params = []interface{}{txCode}
	}
	data := Payload{
		"2.0",
		method,
		params,
		1,
	}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return res, err
	}
	body := bytes.NewReader(payloadBytes)

	resp, err := p.client.Post("https://cloudflare-eth.com", "application/json", body)
	if err != nil {
		return res, err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			fmt.Println("Body Closing error:", err)
		}
	}()
	var result map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return res, err
	}

	errorMessage, errorExist := result["error"]
	if errorExist {
		return res, fmt.Errorf("%s", errorMessage)
	}
	resultBody, resultExist := result["result"]
	if !resultExist {
		return res, fmt.Errorf("%s", "Response does not contain \"result\" field")
	}
	if resultBody == nil {
		return TransactionStructure{}, fmt.Errorf("%s %s %s %s %s", "Transaction", txCode, "Or block", blockNum, "does not exist")
	}
	if callByBlockNumber {
		return p.getTransactionFromBlock(resultBody.(map[string]interface{}), txCode)
	} else {
		//return getTransactionFromBlock(resultBody.(map[string]interface{}), txCode)
		return p.checkTransaction(blockNum, resultBody.(map[string]interface{}))
	}

}

func (p ProxyCache) getTransactionFromBlock(block BlockStructure, txCode string) (TransactionStructure, error) {
	transactions := block.getTransactions()
	if !isHash(txCode) {
		txIndex := new(big.Int)
		txIndex.SetString(txCode, 0)
		if txIndex.IsInt64() && txIndex.Int64() < int64(len(transactions)) {
			tx := TransactionStructure(transactions[txIndex.Int64()].(map[string]interface{}))
			if tx.getTransactionIndex() == txCode {
				p.debugPrinter("Got transaction from block by index")
				return tx, nil
			}
		}
		// else we give a chance to txCode be a txHash, so check it via loop
	}
	for _, transaction := range transactions {
		tx := TransactionStructure(transaction.(map[string]interface{}))
		if tx.getHash() == txCode || tx.getTransactionIndex() == txCode {
			return tx, nil
		}
	}
	return TransactionStructure{}, fmt.Errorf("%s %s", "Transaction not found in the block", block.getNumber())
}

func (p ProxyCache) checkTransaction(blockNum string, tx TransactionStructure) (TransactionStructure, error) {
	if tx.getBlockNum() == blockNum {
		p.debugPrinter("Transaction belongs to block", blockNum)
		return tx, nil
	}
	return TransactionStructure{}, fmt.Errorf("transaction %s does not belong to blockNumber %s", tx.getHash(), blockNum)
}

func (p *ProxyCache) Get(blockNum, txCode string) string {
	p.debugPrinter("We should get blockNum", blockNum, "transaction", txCode)
	var txMapKey string
	if isHash(txCode) {
		txMapKey = txCode
	} else { // argument is index
		txMapKey = blockNum + "-" + txCode
	}
	if tx, txCached := p.txMap[txMapKey]; txCached {
		if tx.tx.getBlockNum() != blockNum {
			return fmt.Sprint("GET: Error occurred: transaction ", txCode, " does not belong to blockNum ", blockNum)
		}
		p.debugPrinter("Moving tx", fmt.Sprintf("%v", tx.tx), "usageIndex", fmt.Sprintf("%d", tx.usageIndex),
			"to usageIndex", fmt.Sprintf("%d", p.usageIndex))
		delete(p.usageIndexMap, tx.usageIndex)
		tx.usageIndex = p.usageIndex
		p.usageIndexMap[tx.usageIndex] = txMapKey
		p.usageIndex++
		return fmt.Sprintf("%v", tx.tx)
	}
	if p.maxTxs > 0 && p.cachedTxs == p.maxTxs {
		p.removeLessUsedTx()
	}
	resTx, err := p.sendRequestForTransaction(blockNum, txCode)
	if err != nil {
		return fmt.Sprint("GET: Error occurred: ", err)
	}
	stringRepresentationOfTx := fmt.Sprintf("%v", resTx)
	for p.maxSize > 0 && (p.cachedSize+uint(len(stringRepresentationOfTx)) > p.maxSize) {
		// TODO would be nice to optimize
		p.removeLessUsedTx()
	}
	if blockNum != "latest" {
		p.addTx(stringRepresentationOfTx, resTx)
	}
	return stringRepresentationOfTx
}

func (p *ProxyCache) addTx(stringRepresentationOfTx string, tx TransactionStructure) {
	if (p.maxSize > 0 && uint(len(stringRepresentationOfTx))+p.cachedSize > p.maxSize) || (p.maxTxs > 0 && p.cachedTxs >= p.maxTxs) {
		fmt.Println("Max cache size reached, ignoring adding", tx)
		return
	}
	p.usageIndexMap[p.usageIndex] = tx.getHash()
	p.txMap[tx.getHash()] = &Transaction{
		p.usageIndex,
		tx,
	}
	p.txMap[tx.getBlockNumTransactionIndexKey()] = p.txMap[tx.getHash()]
	p.cachedSize += uint(len(stringRepresentationOfTx))
	p.cachedTxs++
	p.debugPrinter("Added tx", stringRepresentationOfTx, "with usageIndex", fmt.Sprintf("%d", p.usageIndex))
	p.usageIndex++
}

func (p *ProxyCache) removeLessUsedTx() {
	if p.cachedTxs == 0 {
		return
	}
	minKey := ^uint(0)
	// TODO: try to optimize later
	for key := range p.usageIndexMap {
		if key < minKey {
			minKey = key
		}
	}
	p.cachedSize -= uint(len(fmt.Sprintf("%v", p.txMap[p.usageIndexMap[minKey]].tx)))
	p.cachedTxs--
	p.debugPrinter("Removing least used tx", p.usageIndexMap[minKey], "with index", fmt.Sprintf("%d", minKey))
	tx := p.txMap[p.usageIndexMap[minKey]]
	delete(p.txMap, tx.tx.getHash())
	delete(p.txMap, tx.tx.getBlockNumTransactionIndexKey())
	delete(p.usageIndexMap, minKey)
}

func (p ProxyCache) printCache() string {
	return "----- PrintCache:\ntxMap: " + fmt.Sprintf("%v", p.txMap) +
		"\nusageIndexMap: " + fmt.Sprintf("%v", p.usageIndexMap) +
		"\ncachedTxs " + fmt.Sprintf("%d", p.cachedTxs) +
		"\ncachedSize: " + fmt.Sprintf("%d", p.cachedSize) +
		"\nusageIndex: " + fmt.Sprintf("%d", p.usageIndex)
}

func clearFromEmptyStrings(input []string) []string {
	output := make([]string, 0, len(input))
	for _, s := range input {
		if s != "" {
			output = append(output, s)
		}
	}
	return output
}

func (p *ProxyCache) parseInput(input string) (result string, keepHandling bool) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Errorf("got panic: %v", err)
		}
	}()
	input = strings.Replace(input, "\n", "", -1)
	if strings.Compare(input, "") == 0 {
		result = "Exiting..."
		return
	}
	keepHandling = true
	fmt.Println("->", input)
	splitInput := clearFromEmptyStrings(strings.Split(input, " "))
	if len(splitInput) == 0 {
		result = "Received input of spaces, for exit pass empty string"
		return result, true
	}
	switch strings.ToUpper(splitInput[0]) {
	case "GET":
		parsedCommand := clearFromEmptyStrings(strings.Split(splitInput[1], "/"))
		if len(parsedCommand) != 4 || strings.ToLower(parsedCommand[0]) != "block" || strings.ToLower(parsedCommand[2]) != "tx" {
			result = "received incorrect input: " +
				fmt.Sprintf("%v", splitInput[1]) + " usage: /block/<>/tx/<>"
		} else {
			result = p.Get(parsedCommand[1], parsedCommand[3])
		}
	case PrintCache:
		result = p.printCache()
	default:
		result = "Received unknown command: " + splitInput[0]
	}
	return
}

func main() {
	maxCachedTxsPtr := flag.Uint("t", 0, "Max amount of txs cached by utility, 0 means unlimited")
	maxCacheSizePtr := flag.Uint("s", 0, "Max size of cache in MB, 0 means unlimited")
	printDebug := flag.Bool("d", false, "Print debug messages")
	flag.Parse()
	fmt.Println("Starting cache proxy with max-cached-txs =", *maxCachedTxsPtr, "; max-cache-size =", *maxCacheSizePtr, "; debugPrint=", *printDebug)
	fmt.Println("--- For exit enter empty string ---")
	fmt.Println("Supported commands:")
	fmt.Println("--- GET /block/<blockNum>/tx/<txNum>")
	fmt.Println("---", PrintCache)

	debugPrinter := func(s ...string) {
		// works a bit nasty, but seems fine for debugging
		if *printDebug {
			fmt.Println("DEBUG: ", s)
		}
	}
	reader := bufio.NewReader(os.Stdin)
	proxyCache := ProxyCache{
		http.Client{Timeout: 0},
		debugPrinter,
		*maxCachedTxsPtr,
		0,
		*maxCacheSizePtr * 1024,
		0,
		map[string]*Transaction{},
		map[uint]string{},
		0}
	for {
		input, _ := reader.ReadString('\n')
		result, keepHandling := proxyCache.parseInput(input)
		fmt.Println(result)
		if !keepHandling {
			break
		}
	}
}
