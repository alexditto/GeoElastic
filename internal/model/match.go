package model

// MatchRequest is the input to the fuzzy business matcher. Every field is
// optional — the client sends whatever signals it has, and the matcher
// builds its Elasticsearch query only from the fields actually provided.
type MatchRequest struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Address     Address   `json:"address"`
	PhoneNumber string    `json:"phone_number"`
	Location    *GeoPoint `json:"location"`
}

// IsEmpty reports whether the request carries no usable match criteria at
// all — nothing for the matcher to build a query from.
func (r MatchRequest) IsEmpty() bool {
	return r.Name == "" &&
		r.DisplayName == "" &&
		r.PhoneNumber == "" &&
		r.Address.Street == "" &&
		r.Address.City == "" &&
		r.Address.State == "" &&
		r.Address.Zip == "" &&
		r.Location == nil
}
