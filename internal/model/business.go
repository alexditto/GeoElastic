package model

import "time"

// GeoPoint mirrors Elasticsearch's geo_point field shape.
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Address is embedded directly in Business rather than referenced by ID —
// see the mapping design discussion for why Elasticsearch favors
// denormalization over a separate, joined index for a 1:1 relationship.
type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
	State  string `json:"state"`
	Zip    string `json:"zip"`
}

// OpeningHours is one day's schedule. Business.OpeningHours holds one of
// these per day, mapped as Elasticsearch's `nested` type so a query for
// "open Monday 9am" can't match Tuesday's hours.
type OpeningHours struct {
	Day   string `json:"day"`
	Open  string `json:"open"`
	Close string `json:"close"`
}

// Business is a single geo-tagged business record as stored in Elasticsearch.
type Business struct {
	ID             string         `json:"id,omitempty"`
	Name           string         `json:"name"`
	DisplayName    string         `json:"display_name"`
	BusinessStatus string         `json:"business_status"`
	PrimaryType    string         `json:"primary_type"`
	Address        Address        `json:"address"`
	Location       GeoPoint       `json:"location"`
	PhoneNumber    string         `json:"phone_number"`
	SquareFootage  int            `json:"square_footage"`
	Rating         float64        `json:"rating"`
	OpeningDate    time.Time      `json:"opening_date"`
	OpeningHours   []OpeningHours `json:"opening_hours"`
}
