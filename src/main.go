package main

import (
	"flag"
	"fmt"
)

func main() {
	maxCachedBlocksPtr := flag.Int("b", 0, "Max amount of blocks cached by utility, 0 means unlimited")
	maxCacheSizePtr := flag.Int("s", 0, "Max size of cache in MB, 0 means unlimited")
	flag.Parse()
	fmt.Println("Starting cache proxy with max-cached-blocks =", *maxCachedBlocksPtr, "; max-cache-size =", *maxCacheSizePtr)
}
