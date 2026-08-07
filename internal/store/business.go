package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"geoelastic/internal/model"
	"geoelastic/internal/phone"
)

const (
	exactSearchSize = 10 // enough to detect "more than one" without a real page size
	fuzzySearchSize = 5
	geoBoostPivot   = "5km" // distance at which the location proximity boost is half strength

	// phoneLocalBoost weights a matching phone_number_local (area code
	// excluded) far above the other fuzzy "should" clauses — two businesses
	// sharing a central office code + line number is a much stronger match
	// signal than a fuzzy hit on name/address text.
	phoneLocalBoost = 8.0
)

// BusinessHit pairs a Business with the Elasticsearch relevance score it
// was returned with — meaningful only relative to other hits in the same
// search, not as an absolute confidence value.
type BusinessHit struct {
	Business model.Business
	Score    float64
}

// runBusinessSearch executes a raw Elasticsearch query body against the
// businesses alias and decodes the hits, attaching each hit's _id (never
// present in _source) onto its Business.
func (s *ElasticsearchStore) runBusinessSearch(ctx context.Context, body map[string]interface{}, size int) ([]BusinessHit, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encoding search query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(businessAlias),
		s.client.Search.WithBody(bytes.NewReader(encoded)),
		s.client.Search.WithSize(size),
	)
	if err != nil {
		return nil, fmt.Errorf("searching businesses: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("searching businesses: %s", res.Status())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				ID     string         `json:"_id"`
				Score  float64        `json:"_score"`
				Source model.Business `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}

	hits := make([]BusinessHit, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		hit.Source.ID = hit.ID
		hits[i] = BusinessHit{Business: hit.Source, Score: hit.Score}
	}
	return hits, nil
}

func (s *ElasticsearchStore) GetAllBusinesses(ctx context.Context) ([]model.Business, error) {
	hits, err := s.runBusinessSearch(ctx, map[string]interface{}{}, 1000) // Adjust the size as needed
	if err != nil {
		return nil, err
	}

	businesses := make([]model.Business, len(hits))
	for i, hit := range hits {
		businesses[i] = hit.Business
	}
	return businesses, nil
}

func (s *ElasticsearchStore) GetBusinessByID(ctx context.Context, id string) (*model.Business, error) {
	res, err := s.client.Get(
		businessAlias,
		id,
		s.client.Get.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("getting business by ID: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("getting business by ID: %s", res.Status())
	}

	var result struct {
		ID     string         `json:"_id"`
		Source model.Business `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding get response: %w", err)
	}

	result.Source.ID = result.ID
	return &result.Source, nil
}

// CreateBusiness indexes a new business document into the businesses alias
// and returns the document ID Elasticsearch assigned to it. The phone number
// is normalized to digits-only before indexing, and its last 7 digits are
// additionally stored under phone_number_local — see SearchFuzzyBusiness.
func (s *ElasticsearchStore) CreateBusiness(ctx context.Context, b model.Business) (string, error) {
	b.PhoneNumber = phone.Normalize(b.PhoneNumber)

	doc := struct {
		model.Business
		PhoneNumberLocal string `json:"phone_number_local,omitempty"`
	}{
		Business:         b,
		PhoneNumberLocal: phone.Local(b.PhoneNumber),
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("encoding business: %w", err)
	}

	res, err := s.client.Index(
		businessAlias,
		bytes.NewReader(body),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return "", fmt.Errorf("indexing business: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("indexing business: %s", res.Status())
	}

	var indexed struct {
		ID string `json:"_id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&indexed); err != nil {
		return "", fmt.Errorf("decoding index response: %w", err)
	}

	return indexed.ID, nil
}

// SearchExactBusiness looks for businesses matching every field the caller
// provided in req, exactly. It filters against the .keyword sub-fields for
// name/display_name/street (the analyzed text fields aren't exact-match
// friendly) and the already-keyword fields directly for everything else.
// Fields left blank in req are simply not filtered on.
func (s *ElasticsearchStore) SearchExactBusiness(ctx context.Context, req model.MatchRequest) ([]model.Business, error) {
	var filters []map[string]interface{}

	addFilter := func(field, value string) {
		if value != "" {
			filters = append(filters, map[string]interface{}{
				"term": map[string]interface{}{field: value},
			})
		}
	}

	addFilter("name.keyword", req.Name)
	addFilter("display_name.keyword", req.DisplayName)
	addFilter("address.street.keyword", req.Address.Street)
	addFilter("address.city", req.Address.City)
	addFilter("address.state", req.Address.State)
	addFilter("address.zip", req.Address.Zip)
	addFilter("phone_number", phone.Normalize(req.PhoneNumber))

	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filters,
			},
		},
	}

	hits, err := s.runBusinessSearch(ctx, body, exactSearchSize)
	if err != nil {
		return nil, err
	}

	businesses := make([]model.Business, len(hits))
	for i, hit := range hits {
		businesses[i] = hit.Business
	}
	return businesses, nil
}

// SearchFuzzyBusiness ranks businesses by how well they match whatever
// fields the caller provided in req, tolerating typos on the analyzed text
// fields (name, display_name, address.street) via fuzziness, treating the
// keyword-only fields (city/state/zip) as exact-or-nothing signals, and
// boosting results near req.Location if it was provided. Phone numbers are
// compared on their last 7 digits (phone_number_local) rather than the
// full number — that tolerates format differences and heavily boosts the
// score when it matches, since a shared area code alone (common to many
// unrelated businesses in the same region) isn't a meaningful signal on
// its own and is deliberately not scored at all.
func (s *ElasticsearchStore) SearchFuzzyBusiness(ctx context.Context, req model.MatchRequest) ([]BusinessHit, error) {
	var should []map[string]interface{}

	addFuzzyMatch := func(field, value string) {
		if value != "" {
			should = append(should, map[string]interface{}{
				"match": map[string]interface{}{
					field: map[string]interface{}{
						"query":     value,
						"fuzziness": "AUTO",
					},
				},
			})
		}
	}
	addTerm := func(field, value string) {
		if value != "" {
			should = append(should, map[string]interface{}{
				"term": map[string]interface{}{field: value},
			})
		}
	}

	addFuzzyMatch("name", req.Name)
	addFuzzyMatch("display_name", req.DisplayName)
	addFuzzyMatch("address.street", req.Address.Street)
	addTerm("address.city", req.Address.City)
	addTerm("address.state", req.Address.State)
	addTerm("address.zip", req.Address.Zip)

	if local := phone.Local(phone.Normalize(req.PhoneNumber)); local != "" {
		should = append(should, map[string]interface{}{
			"term": map[string]interface{}{
				"phone_number_local": map[string]interface{}{
					"value": local,
					"boost": phoneLocalBoost,
				},
			},
		})
	}

	if req.Location != nil {
		should = append(should, map[string]interface{}{
			"distance_feature": map[string]interface{}{
				"field": "location",
				"pivot": geoBoostPivot,
				"origin": map[string]interface{}{
					"lat": req.Location.Lat,
					"lon": req.Location.Lon,
				},
			},
		})
	}

	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should":               should,
				"minimum_should_match": 1,
			},
		},
	}

	return s.runBusinessSearch(ctx, body, fuzzySearchSize)
}
