package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"
)

func main() {
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
		candidates = append(candidates, item)
	}
	slices.SortFunc(candidates, func(a, b bzItem) int {
		return b.Delta - a.Delta
	})
	for _, item := range candidates {
		margin := item.BuyPrice - item.SellPrice
		marginPercent := (margin / item.BuyPrice) * 100
		fmt.Printf("%s — margin: %.0f (%.1f%%) — 60 second item flow: %d\n", item.Name, margin, marginPercent, item.Delta)
	}
}

func fetchItems() ([]bzItem, error) {
	resp, err := http.Get("https://api.skyblock.bz/api/all")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var items []bzItem
	err = json.Unmarshal(body, &items)
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
