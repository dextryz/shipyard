package main

import (
	"fmt"
	"os"

	"github.com/dextryz/shipyard"
)

func main() {
	err := shipyard.Main()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
