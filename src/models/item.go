package models

import "time"

type Item struct {
	Base
	Title       string     `json:"title"`
	Amount      float64    `json:"amount"`
	Unit        string     `json:"unit"`
	Bought      bool       `json:"bought"`
	Shop        string     `json:"shop"`
	Description string     `json:"description"`
	When        *time.Time `json:"when"`
}

type ItemWithID struct {
	Item
	ID string `json:"id"`
}

type ItemSearchResult struct {
	Item
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}
