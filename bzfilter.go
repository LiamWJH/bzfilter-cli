package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"math/rand/v2"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("[-1] for random item\n[0] for item search\n[1] for data refresh\n[2] for filtering\n[3] for exit\n[4] for filter setting\n")
		var command string
		command, _ = reader.ReadString('\n')
		command = strings.TrimSpace(command)

		cfg := load_config()
		switch command {
		case "-1" :
			firstPoll, _ := read_cache("firstPoll.json")
			secondPoll, _ := read_cache("secondPoll.json")
			firstMap := make(map[string]bzItem)
			for _, item := range firstPoll {
				firstMap[item.Id] = item
			}
			keys := make([]string, 0, len(firstMap))

			for k := range firstMap {
				keys = append(keys, k)
			}
			itemName := keys[rand.IntN(len(keys))]
			var found *bzItem
			for i, item := range secondPoll {
				if item.Name == itemName || item.Id == itemName {
					first, ok := firstMap[item.Id]
					if ok {
						secondPoll[i].BuyDelta = item.BuyTrans - first.BuyTrans
						secondPoll[i].SellDelta = item.SellTrans - first.SellTrans
						secondPoll[i].Delta = secondPoll[i].BuyDelta + secondPoll[i].SellDelta
						secondPoll[i].Margin = item.BuyPrice - item.SellPrice
						secondPoll[i].MarginPercent = (secondPoll[i].Margin / item.BuyPrice) * 100
					}
					found = &secondPoll[i]
					break
				}
			}
			if found == nil {
				fmt.Println("Item not found")
			} else {
				fmt.Printf("Item: %s\nprice: %.0f\nmargin: %.0f\nmargin%%: %.1f\nbuy volume: %d\nsell volume: %d\nbuy delta: %d\nsell delta: %d\ndelta: %d\n",
					found.Name, found.BuyPrice, found.Margin, found.MarginPercent,
					found.BuyVolume, found.SellVolume,
					found.BuyDelta, found.SellDelta, found.Delta,
				)
			}
		case "0":
			fmt.Printf("Search: ")
			itemName, _ := reader.ReadString('\n')
			itemName = strings.TrimSpace(itemName)
			firstPoll, _ := read_cache("firstPoll.json")
			secondPoll, _ := read_cache("secondPoll.json")
			firstMap := make(map[string]bzItem)
			for _, item := range firstPoll {
				firstMap[item.Id] = item
			}
			var found *bzItem
			for i, item := range secondPoll {
				if item.Name == itemName || item.Id == itemName {
					first, ok := firstMap[item.Id]
					if ok {
						secondPoll[i].BuyDelta = item.BuyTrans - first.BuyTrans
						secondPoll[i].SellDelta = item.SellTrans - first.SellTrans
						secondPoll[i].Delta = secondPoll[i].BuyDelta + secondPoll[i].SellDelta
						secondPoll[i].Margin = item.BuyPrice - item.SellPrice
						secondPoll[i].MarginPercent = (secondPoll[i].Margin / item.BuyPrice) * 100
					}
					found = &secondPoll[i]
					break
				}
			}
			if found == nil {
				fmt.Println("Item not found")
			} else {
				fmt.Printf("Item: %s\nprice: %.0f\nmargin: %.0f\nmargin%%: %.1f\nbuy volume: %d\nsell volume: %d\nbuy delta: %d\nsell delta: %d\ndelta: %d\n",
					found.Name, found.BuyPrice, found.Margin, found.MarginPercent,
					found.BuyVolume, found.SellVolume,
					found.BuyDelta, found.SellDelta, found.Delta,
				)
			}
		case "1":
			refresh_pull(cfg.PollTime)
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
				if item.BuyPrice-item.SellPrice < cfg.MinMargin {
					continue
				}
				avgBuyVolume := (first.BuyVolume + item.BuyVolume) / 2
				avgSellVolume := (first.SellVolume + item.SellVolume) / 2
				if avgBuyVolume < cfg.MinBuyVolume || avgSellVolume < cfg.MinSellVolume {
					continue
				}
				if item.BuyPrice > cfg.MaxPrice {
					continue
				}
				margin := item.BuyPrice - item.SellPrice
				marginPercent := (margin / item.BuyPrice) * 100
				if marginPercent < cfg.MinMarginPercent {
					continue
				}
				item.BuyDelta = item.BuyTrans - first.BuyTrans
				item.SellDelta = item.SellTrans - first.SellTrans
				item.Delta = item.BuyDelta + item.SellDelta
				if item.BuyDelta < cfg.MinBuyDelta || item.SellDelta < cfg.MinSellDelta || item.BuyDelta == 0 || item.SellDelta == 0 {
					continue
				}
				if item.Delta <= 0 {
					continue
				}
				item.Margin = margin
				item.MarginPercent = marginPercent
				candidates = append(candidates, item)
			}
			slices.SortFunc(candidates, func(a, b bzItem) int {
				switch cfg.SortFrom {
				case "margin":
					if b.Margin > a.Margin {
						return 1
					}
					if b.Margin < a.Margin {
						return -1
					}
					return 0
				case "margin%":
					if b.MarginPercent > a.MarginPercent {
						return 1
					}
					if b.MarginPercent < a.MarginPercent {
						return -1
					}
					return 0
				case "buy_price":
					if b.BuyPrice > a.BuyPrice {
						return 1
					}
					if b.BuyPrice < a.BuyPrice {
						return -1
					}
					return 0
				default:
					if b.Delta > a.Delta {
						return 1
					}
					if b.Delta < a.Delta {
						return -1
					}
					return 0
				}
			})
			if len(candidates) != 0 {
				for _, item := range candidates {
					fmt.Printf("%s — margin: %.0f (%.1f%%) — buy flow: %d sell flow: %d\n", item.Name, item.Margin, item.MarginPercent, item.BuyDelta, item.SellDelta)
				}
			} else {
				fmt.Println("NO MATCHING ITEM")
			}
		case "3":
			os.Exit(0)
		case "4":
			cfg := load_config()
			fmt.Println("input a value or 'skip' to keep current")
			var input string
			fmt.Printf("poll time? (current: %d) ", cfg.PollTime)
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input != "skip" {
				if val, err := strconv.Atoi(input); err == nil {
					cfg.PollTime = val
				}
			}

			fmt.Printf("minimum margin? (current: %.0f) ", cfg.MinMargin)
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "skip" {
				if val, err := strconv.ParseFloat(input, 64); err == nil {
					cfg.MinMargin = val
				}
			}

			fmt.Printf("minimum buy volume? (current: %d) ", cfg.MinBuyVolume)
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "skip" {
				if val, err := strconv.Atoi(input); err == nil {
					cfg.MinBuyVolume = val
				}
			}

			fmt.Printf("minimum sell volume? (current: %d) ", cfg.MinSellVolume)
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "skip" {
				if val, err := strconv.Atoi(input); err == nil {
					cfg.MinSellVolume = val
				}
			}

			fmt.Printf("minimum buy %d second amount (current %d) ", cfg.PollTime, cfg.MinBuyDelta)
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "skip" {
				if val, err := strconv.Atoi(input); err == nil {
					cfg.MinBuyDelta = val
				}
			}
			fmt.Printf("minimum sell %d second amount (current %d) ", cfg.PollTime, cfg.MinSellDelta)
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "skip" {
				if val, err := strconv.Atoi(input); err == nil {
					cfg.MinSellDelta = val
				}
			}
			fmt.Printf("max price? (current: %.0f) ", cfg.MaxPrice)
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "skip" {
				if val, err := strconv.ParseFloat(input, 64); err == nil {
					cfg.MaxPrice = val
				}
			}

			fmt.Printf("minimum margin%%? (current: %.0f) ", cfg.MinMarginPercent)
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "skip" {
				if val, err := strconv.ParseFloat(input, 64); err == nil {
					cfg.MinMarginPercent = val
				}
			}

			fmt.Printf("sort from [moving item/margin/margin%%/buy_price]? (current: %s) ", cfg.SortFrom)
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "skip" {
				cfg.SortFrom = input
			}

			save_config(cfg)
			fmt.Println("config saved!")
		default:
			fmt.Println("Not a valid option")
		}
	}
}

func refresh_pull(pollTime int) {
	fmt.Println("First poll...")
	firstPoll, err := fetchItems()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%v seconds...(insta buy/insta sell accuracy checks)\n", pollTime)
	time.Sleep(time.Duration(pollTime) * time.Second)
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

type filterConfig struct {
	PollTime         int     `json:"poll_time"`
	MinMargin        float64 `json:"min_margin"`
	MinBuyVolume     int     `json:"min_buy_volume"`
	MinSellVolume    int     `json:"min_sell_volume"`
	MinBuyDelta      int     `json:"min_buy_delta"`
	MinSellDelta     int     `json:"min_sell_delta"`
	MaxPrice         float64 `json:"max_price"`
	MinMarginPercent float64 `json:"min_margin_percent"`
	SortFrom         string  `json:"sort_from"`
}

func load_config() filterConfig {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return filterConfig{
			PollTime:         60,
			MinMargin:        500,
			MinBuyVolume:     50000,
			MinSellVolume:    50000,
			MinBuyDelta:      150,
			MinSellDelta:     150,
			MaxPrice:         3000000,
			MinMarginPercent: 10,
			SortFrom:         "moving item",
		}
	}
	var cfg filterConfig
	json.Unmarshal(data, &cfg)
	return cfg
}

func save_config(cfg filterConfig) {
	data, _ := json.Marshal(cfg)
	os.WriteFile("config.json", data, 0644)
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
	Name          string  `json:"name"`
	Id            string  `json:"id"`
	BuyPrice      float64 `json:"buy_price"`
	SellPrice     float64 `json:"sell_price"`
	BuyVolume     int     `json:"buy_volume"`
	SellVolume    int     `json:"sell_volume"`
	BuyTrans      int     `json:"buy_trans"`
	SellTrans     int     `json:"sell_trans"`
	Icon          string  `json:"icon"`
	Visits        int     `json:"visits"`
	Delta         int
	BuyDelta      int
	SellDelta     int
	Margin        float64
	MarginPercent float64
}
