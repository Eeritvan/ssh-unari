package fetch

import (
	"encoding/json"
	"io"
	"net/http"
)

const UNICAFE_API = "https://unicafe.fi/wp-json/swiss/v1/restaurants/?lang=fi"

type Unicafe struct {
	Id            uint       `json:"id"`
	Title         string     `json:"title"`
	Slug          string     `json:"slug"`
	Permalink     string     `json:"permalink"`
	Location      []Location `json:"location"`
	Address       string     `json:"address"`
	VisitingHours string     `json:"visitingHours"`
	Menu          MenuData   `json:"menuData"`
	// BlocksHtml    string     `json:"blockHtml"`
}

type Location struct {
	Id   uint   `json:"id"`
	Name string `json:"name"`
}

type MenuData struct {
	Id              uint          `json:"id"`
	Email           string        `json:"email"`
	Phone           string        `json:"phone"`
	Address         string        `json:"address"`
	FeedbackAddress string        `json:"feedback_address"`
	Description     string        `json:"description"`
	VisitingHours   VisitingHours `json:"visitingHours"`
	Menus           []Menu        `json:"menus"`
	Name            string        `json:"name"`
	Areacode        uint          `json:"areacode"`
}

type VisitingHours struct {
	Business  any `json:"business"`  // TODO: any type
	Breakfast any `json:"breakfast"` // TODO: any type
	Bistro    any `json:"bistro"`    // TODO: any type
	Lunch     any `json:"lounas"`    // TODO: any type
}

type Menu struct {
	Date    string `json:"date"`
	Message string `json:"message"`
	Data    []Data `json:"data"`
}

type Data struct {
	Name        string `json:"name"`
	Ingredients string `json:"ingredients"`
	Nutrition   string `json:"nutrition"`
	Price       Price  `json:"price"`
	// Meta        any    `json:"meta"`
}

type Price struct {
	Value any    `json:"value"` // TODO: any type
	Name  string `json:"name"`
}

func GetUnicafe() ([]Unicafe, error) {
	resp, err := http.Get(UNICAFE_API)
	if err != nil {
		// TODO: better error message
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)

	var data []Unicafe
	err = json.Unmarshal(body, &data)
	if err != nil {
		// TODO: better error message
		return nil, err
	}
	return data, nil
}
