package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"time"
)

func main() {
	for {
		fmt.Print("[1] for data refresh\n[2] for filtering\n[3] for exit\n")
		var command string
		fmt.Scan(&command)
		switch command {
		case "1":
			refresh_pull()
		case "2":
			firstPoll, _ := read_cache("firstPoll.json")
			secondPoll, _ := read_cache("secondPoll.json")
			firstMap := make(map[string]bzItem)
			for _, item := range firstPoll {
				firstMap[item.Id] = item
			}
			var candidates []bzItem
			for _, item := range secondPoll {
				first, ok := firstMap[item.Id]
				if !ok {
					continue
				}
				if item.BuyPrice-item.SellPrice < 500 {
					continue
				}
				avgBuyVolume := (first.BuyVolume + item.BuyVolume) / 2
				avgSellVolume := (first.SellVolume + item.SellVolume) / 2
				if avgBuyVolume < 50000 || avgSellVolume < 50000 {
					continue
				}
				if item.BuyPrice > 3000000 {
					continue
				}
				margin := item.BuyPrice - item.SellPrice
				marginPercent := (margin / item.BuyPrice) * 100
				if marginPercent < 10 {
					continue
				}
				item.Delta = (item.BuyTrans + item.SellTrans) - (first.BuyTrans + first.SellTrans)
				if item.Delta <= 0 {
					continue
				}
				candidates = append(candidates, item)
			}
			slices.SortFunc(candidates, func(a, b bzItem) int {
				if b.Delta > a.Delta {
					return 1
				}
				if b.Delta < a.Delta {
					return -1
				}
				return 0
			})
			for _, item := range candidates {
				margin := item.BuyPrice - item.SellPrice
				marginPercent := (margin / item.BuyPrice) * 100
				fmt.Printf("%s — margin: %.0f (%.1f%%) — 60 second item flow: %d\n", item.Name, margin, marginPercent, item.Delta)
			}
		case "3":
			os.Exit(0)
		default:
			fmt.Println("Not a valid option")
		}
	}
}

func refresh_pull() {
	fmt.Println("First poll...")
	firstPoll, err := fetchItems()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("60 seconds...(insta buy/insta sell accuracy checks)")
	time.Sleep(60 * time.Second)
	fmt.Println("Second poll...")
	secondPoll, err := fetchItems()
	if err != nil {
		fmt.Println(err)
		return
	}
	store_cache("firstPoll.json", firstPoll)
	store_cache("secondPoll.json", secondPoll)
}

func fetchItems() ([]byte, error) {
	resp, err := http.Get("https://api.skyblock.bz/api/all")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

func store_cache(filename string, data []byte) {
	os.WriteFile(filename, data, 0644)
}

func read_cache(filename string) ([]bzItem, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
	}
	var items []bzItem
	err = json.Unmarshal(data, &items)
	return items, err
}

type bzItem struct {
	Name       string  `json:"name"`
	Id         string  `json:"id"`
	BuyPrice   float64 `json:"buy_price"`
	SellPrice  float64 `json:"sell_price"`
	BuyVolume  int     `json:"buy_volume"`
	SellVolume int     `json:"sell_volume"`
	BuyTrans   int     `json:"buy_trans"`
	SellTrans  int     `json:"sell_trans"`
	Icon       string  `json:"icon"`
	Visits     int     `json:"visits"`
	Delta      int
}