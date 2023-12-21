package models

type Item struct {
	Base
	Title  string `json:"title"`
	Amount int    `json:"amount"`
	Unit   string `json:"unit"`
	Bought bool   `json:"bought"`
}

type SearchResult[T any] struct {
	Data  T       `json:"data"`
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}
