package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

const PrintCache = "PRINTCACHE"

type Transaction struct {
	usageIndex uint
	tx         string // in json format
}

type ProxyCache struct {
	maxTxs        uint
	cachedTxs     uint
	maxSize       uint
	cachedSize    uint
	txMap         map[string] /*hash*/ Transaction
	usageIndexMap map[uint]string
	usageIndex    uint
}

func (p *ProxyCache) Get(block, txHash string) string {
	fmt.Println("We should get block", block, "transaction", txHash)
	return ""
}

func (p ProxyCache) printCache() {
	fmt.Println("PrintCache: ", p)
}

func (p *ProxyCache) parseInput(input string) (keepHandling bool) {
	input = strings.Replace(input, "\n", "", -1)
	if strings.Compare(input, "") == 0 {
		fmt.Println("Exiting...")
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
			fmt.Println("received incorrect input:", parsedCommand, " usage: /block/<>/tx/<>")
			return
		}
		p.Get(parsedCommand[1], parsedCommand[3])
	case PrintCache:
		p.printCache()
	default:
		fmt.Println("Received unknown command:", splitInput[0])
	}
	return
}

func main() {
	maxCachedTxsPtr := flag.Uint("b", 0, "Max amount of txs cached by utility, 0 means unlimited")
	maxCacheSizePtr := flag.Uint("s", 0, "Max size of cache in MB, 0 means unlimited")
	flag.Parse()
	fmt.Println("Starting cache proxy with max-cached-txs =", *maxCachedTxsPtr, "; max-cache-size =", *maxCacheSizePtr)
	fmt.Println("--- For exit enter empty string ---")
	fmt.Println("Supported commands:")
	fmt.Println("--- GET /block/<blockNum>/tx/<txNum>")
	fmt.Println("---", PrintCache)

	reader := bufio.NewReader(os.Stdin)
	proxyCache := ProxyCache{
		*maxCachedTxsPtr,
		0,
		*maxCacheSizePtr,
		0,
		map[string]Transaction{},
		map[uint]string{},
		0}
	for {
		input, _ := reader.ReadString('\n')
		if !proxyCache.parseInput(input) { // logging is inside
			break
		}
	}
}
