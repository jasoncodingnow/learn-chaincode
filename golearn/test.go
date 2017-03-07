package main

import (
	"fmt"
)

func main() {
	// trades := AllTrades{}
	// for i := 0; i < 10; i++ {
	// 	var trade = AnOpenTrade{}
	// 	trades.OpenTrades = append(trades.OpenTrades, trade)
	// }
	// fmt.Println(trades)
	trades := []int{1, 2}
	for i := 0; i < 10; i++ {
		trades = append(trades, i*10)
	}
	fmt.Println(trades)
	fmt.Println(trades[:4])
	trades = append(trades[:4], trades[5:]...)
	fmt.Println(trades)
}
