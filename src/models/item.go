package models

type Item struct {
	Base
	Title  string  `json:"title"`
	Amount float64 `json:"amount,default=1"`
	Unit   string  `json:"unit,default=pc"`
	Bought bool    `json:"bought,default=false"`
	Shop   string  `json:"shop,default=Edeka"`
}

type ItemWithId struct {
	Item
	ID string `json:"id"`
}

type ItemSearchResult struct {
	Item
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}
