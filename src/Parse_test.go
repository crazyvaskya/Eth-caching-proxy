package main

import (
	"fmt"
	"net/http"
	"testing"
)

func TestProxyCache(t *testing.T) {
	proxyCache := ProxyCache{
		http.Client{Timeout: 0},
		Printer(func(...string) {}),
		0,
		0,
		0,
		0,
		map[string]*Transaction{},
		map[uint]string{},
		0}

	res, exit := proxyCache.parseInput("")
	expectedRes := "Exiting..."
	if res != expectedRes || exit != false {
		t.Errorf("Parse empty string failed, expected: [%s, %t], got [%s, %t]", expectedRes, true, res, exit)
	}
	res, exit = proxyCache.parseInput(" ")
	expectedRes = "Received input of spaces, for exit pass empty string"
	if res != expectedRes || exit != true {
		t.Errorf("Parse space string failed, expected: [%s, %t], got [%s, %t]", expectedRes, true, res, exit)
	}
	res, exit = proxyCache.parseInput(" printcache   ")
	expectedRes = "----- PrintCache:\ntxMap: map[]" +
		"\nusageIndexMap: map[]" +
		"\ncachedTxs 0" +
		"\ncachedSize: 0" +
		"\nusageIndex: 0"
	if res != expectedRes || exit != true {
		t.Errorf("Parse PRINTCACHE failed, expected: [%s, %t], got [%s, %t]", expectedRes, true, res, exit)
	}
	res, exit = proxyCache.parseInput("get /haha/0x123/tx/hash")
	expectedRes = "received incorrect input: /haha/0x123/tx/hash usage: /block/<>/tx/<>"
	if res != expectedRes || exit != true {
		t.Errorf("Parse space string failed, expected: [%s, %t], got [%s, %t]", expectedRes, true, res, exit)
	}
	res, exit = proxyCache.parseInput("get /haha /0x123/tx/hash")
	expectedRes = "received incorrect input: /haha usage: /block/<>/tx/<>"
	if res != expectedRes || exit != true {
		t.Errorf("Parse space string failed, expected: [%s, %t], got [%s, %t]", expectedRes, true, res, exit)
	}
	res, exit = proxyCache.parseInput("get /block/0x123/tax/hash")
	expectedRes = "received incorrect input: /block/0x123/tax/hash usage: /block/<>/tx/<>"
	if res != expectedRes || exit != true {
		t.Errorf("Parse space string failed, expected: [%s, %t], got [%s, %t]", expectedRes, true, res, exit)
	}
}

func TestProxyCache_Get(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("Got panic: %v", err)
		}
	}()
	proxyCache := ProxyCache{
		debugPrinter:  func(s ...string) {},
		txMap:         map[string]*Transaction{},
		usageIndexMap: map[uint]string{},
	}
	res, _ := proxyCache.parseInput("get /block/0xa4fb72/tx/0x2")
	expectedRes := "map[blockHash:0x4d5927b33797eb7104860c9704f34b0916f4874c143e819947e5475b68fdbc9c blockNumber:0xa4fb72 from:0x2b09e5896f3506d378f40df30faa4b0c25aabc2f gas:0x13d620 gasPrice:0x48690b6600 hash:0x4902d4cc958e0a8138cddbd1845373a5c45504ff7d385f3005394b7b0feea978 input:0xc89e4361f2cac36da45d027401539f999e887c7ca0bf0a1790060e32e2156ec17f3131e9c0bad196d65fac95f31db1acd4e559f9b86e6073a564a6a200000000541a1f10bddb3502ca7794de311511a6091e0c7f1fc5c40000f324030000004d55000000eddda06a90ffaaea3db0c308f29f5798b7de432555000000df8bb892d5d730a759744bf717560171acc5a69c55000001 nonce:0xf13 r:0x4928c513c3b3ff737dcd717f8693fa4b3742cee59be739af67ff2313c55bf4d5 s:0x15334eb53ad39f0b5e05e7e78025c5cc5e96a30a2f0ffd5394cda7fb76e6b11a to:0x78a55b9b3bbeffb36a43d9905f654d2769dc55e8 transactionIndex:0x2 v:0x1b value:0x0]"
	if res != expectedRes {
		t.Errorf("Parse GET failed, expected: \"%s\", got \"%s\"", expectedRes, res)
	}

	// the same tx but with hash
	res, _ = proxyCache.parseInput("get /block/0xa4fb72/tx/0x4902d4cc958e0a8138cddbd1845373a5c45504ff7d385f3005394b7b0feea978")
	expectedRes = "map[blockHash:0x4d5927b33797eb7104860c9704f34b0916f4874c143e819947e5475b68fdbc9c blockNumber:0xa4fb72 from:0x2b09e5896f3506d378f40df30faa4b0c25aabc2f gas:0x13d620 gasPrice:0x48690b6600 hash:0x4902d4cc958e0a8138cddbd1845373a5c45504ff7d385f3005394b7b0feea978 input:0xc89e4361f2cac36da45d027401539f999e887c7ca0bf0a1790060e32e2156ec17f3131e9c0bad196d65fac95f31db1acd4e559f9b86e6073a564a6a200000000541a1f10bddb3502ca7794de311511a6091e0c7f1fc5c40000f324030000004d55000000eddda06a90ffaaea3db0c308f29f5798b7de432555000000df8bb892d5d730a759744bf717560171acc5a69c55000001 nonce:0xf13 r:0x4928c513c3b3ff737dcd717f8693fa4b3742cee59be739af67ff2313c55bf4d5 s:0x15334eb53ad39f0b5e05e7e78025c5cc5e96a30a2f0ffd5394cda7fb76e6b11a to:0x78a55b9b3bbeffb36a43d9905f654d2769dc55e8 transactionIndex:0x2 v:0x1b value:0x0]"
	if res != expectedRes {
		t.Errorf("Parse GET failed, expected: \"%s\", got \"%s\"", expectedRes, res)
	}

	// the same tx, wrong block, but Tx is cached
	res, _ = proxyCache.parseInput("get /block/0xb4fb72/tx/0x4902d4cc958e0a8138cddbd1845373a5c45504ff7d385f3005394b7b0feea978")
	expectedRes = "GET: Error occurred: transaction 0x4902d4cc958e0a8138cddbd1845373a5c45504ff7d385f3005394b7b0feea978 does not belong to blockNum 0xb4fb72"
	if res != expectedRes {
		t.Errorf("Parse GET failed, expected: \"%s\", got \"%s\"", expectedRes, res)
	}

	// transaction from another block
	res, _ = proxyCache.parseInput("get /block/0xa4fb72/tx/0xaa8a64e74dab0cd1d08b9d91d523c4647a8b536e08d77c6dd7b455535d4f820e")
	expectedRes = "GET: Error occurred: transaction 0xaa8a64e74dab0cd1d08b9d91d523c4647a8b536e08d77c6dd7b455535d4f820e does not belong to blockNumber 0xa4fb72"
	if res != expectedRes {
		t.Errorf("Parse GET failed, expected: \"%s\", got \"%s\"", expectedRes, res)
	}

	//get latest block this test can be floating
	res, _ = proxyCache.parseInput("get block/latest/tx/0x0")
	fmt.Println("GET: latest block, 0x0 transaction:", res)
	if res[:19] == "GET: Error occurred" {
		t.Errorf("Parse GET latest failed, expected: not empty block, got \"%s\"", res)
	}
}
