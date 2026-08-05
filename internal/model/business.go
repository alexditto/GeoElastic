package model

// GeoPoint mirrors Elasticsearch's geo_point field shape.
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Business is a single geo-tagged business record as stored in Elasticsearch.
type Business struct {
	ID            string   `json:"id,omitempty"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Address       string   `json:"address"`
	Phone         string   `json:"phone"`
	SquareFootage int      `json:"square_footage"`
	Location      GeoPoint `json:"location"`
}
