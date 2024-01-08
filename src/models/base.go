package models

type Base struct {
	Created int64 `json:"created"`
	Updated int64 `json:"updated"`
}

type Total struct {
	Total int `json:"total"`
}

type ID struct {
	ID string `json:"id"`
}
