package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

const PrintCache = "PRINTCACHE"

type Printer func(...string)
type Transaction struct {
	usageIndex uint
	tx         string // in json format
}

type ProxyCache struct {
	maxTxs        uint
	cachedTxs     uint
	maxSize       uint // in bytes
	cachedSize    uint // in bytes
	txMap         map[string] /*hash*/ Transaction
	usageIndexMap map[uint]string
	usageIndex    uint
}


func (p *ProxyCache) Get(block, txHash string) string {
	p.debugPrinter("We should get block", block, "transaction", txHash)
	tx, txCached := p.txMap[txHash]
	if txCached {
		p.debugPrinter("Moving tx", tx.tx, "usageIndex", fmt.Sprintf("%d", tx.usageIndex),
			"to usageIndex", fmt.Sprintf("%d", p.usageIndex))
		delete(p.usageIndexMap, tx.usageIndex)
		tx.usageIndex = p.usageIndex
		p.usageIndexMap[tx.usageIndex] = txHash
		p.txMap[txHash] = tx
		p.usageIndex++
		return tx.tx
	}
	if p.maxTxs > 0 && uint(len(p.txMap)) == p.maxTxs {
		p.removeLessUsedTx()
	}
	testTx := "This is my test Tx with hash " + txHash
	for p.maxSize > 0 && (p.cachedSize+uint(len(testTx)) > p.maxSize) {
		p.removeLessUsedTx()
	}
	p.addTx(txHash, testTx)
	return testTx
}

func (p *ProxyCache) addTx(txHash, tx string) {
	if (p.maxSize > 0 && uint(len(tx))+p.cachedSize > p.maxSize) || (p.maxTxs > 0 && p.cachedTxs >= p.maxTxs) {
		fmt.Println("Max cache size reached, ignoring adding", tx)
		return
	}
	p.usageIndexMap[p.usageIndex] = txHash
	p.txMap[txHash] = Transaction{
		p.usageIndex,
		tx,
	}
	p.cachedSize += uint(len(tx))
	p.cachedTxs++
	p.debugPrinter("Added tx", tx, "with usageIndex", fmt.Sprintf("%d", p.usageIndex))
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
	p.cachedSize -= uint(len(p.txMap[p.usageIndexMap[minKey]].tx))
	p.cachedTxs--
	p.debugPrinter("Removing least used tx", p.txMap[p.usageIndexMap[minKey]].tx, "with index", fmt.Sprintf("%d", minKey))
	delete(p.txMap, p.usageIndexMap[minKey])
	delete(p.usageIndexMap, minKey)
}

func (p ProxyCache) printCache() string {
	return "----- PrintCache:\ntxMap: " + fmt.Sprintf("%v", p.txMap) +
		"\nusageIndexMap: " + fmt.Sprintf("%v", p.usageIndexMap) +
		"\ncachedTxs " + fmt.Sprintf("%d", p.cachedTxs) +
		"\ncachedSize: " + fmt.Sprintf("%d", p.cachedSize) +
		"\nusageIndex: " + fmt.Sprintf("%d", p.usageIndex)
}

func (p *ProxyCache) parseInput(input string) (result string, keepHandling bool) {
	input = strings.Replace(input, "\n", "", -1)
	if strings.Compare(input, "") == 0 {
		result = "Exiting..."
		return
	}
	keepHandling = true
	fmt.Println("->", input)
	splitInput := strings.Split(input, " ")

	switch strings.ToUpper(splitInput[0]) {
	case "GET":
		parsedCommand := strings.Split(splitInput[1], "/")
		if parsedCommand[0] == "" {
			parsedCommand = parsedCommand[1:]
		}
		if len(parsedCommand) != 4 || strings.ToLower(parsedCommand[0]) != "block" || strings.ToLower(parsedCommand[2]) != "tx" {
			result = "received incorrect input: " +
				fmt.Sprintf("%v", parsedCommand) + " usage: /block/<>/tx/<>"
		}
		result = p.Get(parsedCommand[1], parsedCommand[3])
	case PrintCache:
		result = p.printCache()
	default:
		result = "Received unknown command: " + splitInput[0]
	}
	return
}

func main() {
	maxCachedTxsPtr := flag.Uint("b", 0, "Max amount of txs cached by utility, 0 means unlimited")
	maxCacheSizePtr := flag.Uint("s", 0, "Max size of cache in MB, 0 means unlimited")
	printDebug := flag.Bool("d", false, "Print debug messages")
	flag.Parse()
	fmt.Println("Starting cache proxy with max-cached-txs =", *maxCachedTxsPtr, "; max-cache-size =", *maxCacheSizePtr, "; debugPrint=", *printDebug)
	fmt.Println("--- For exit enter empty string ---")
	fmt.Println("Supported commands:")
	fmt.Println("--- GET /block/<blockNum>/tx/<txNum>")
	fmt.Println("---", PrintCache)

	debugPrinter := func(s ...string) {
		if *printDebug {
			fmt.Println("DEBUG: ", s)
		}
	}
	reader := bufio.NewReader(os.Stdin)
	proxyCache := ProxyCache{
		http.Client{Timeout: 2},
		debugPrinter,
		*maxCachedTxsPtr,
		0,
		*maxCacheSizePtr * 1024,
		0,
		map[string]Transaction{},
		map[uint]string{},
		0}
	for {
		input, _ := reader.ReadString('\n')
		result, keepHandling := proxyCache.parseInput(input)
		fmt.Println(result)
		if !keepHandling { // logging is inside
			break
		}
	}
}
