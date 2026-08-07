package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"geoelastic/internal/config"
	"geoelastic/internal/model"
	"geoelastic/internal/store"
)

func businessName() string {
	names := [20]string{
		"Acme Corporation",
		"Globex Corporation",
		"Soylent Corp",
		"Initech",
		"Umbrella Corporation",
		"Hooli",
		"Stark Industries",
		"Wayne Enterprises",
		"Wonka Industries",
		"Tyrell Corporation",
		"Cyberdyne Systems",
		"Vandelay Industries",
		"Gringotts Wizarding Bank",
		"Monsters, Inc.",
		"Oceanic Airlines",
		"Prestige Worldwide",
		"Bluth Company",
		"Dunder Mifflin",
		"Pied Piper",
		"Good Burger",
	}
	return names[rand.IntN(len(names))]
}

func businessStatus() string {
	statuses := [5]string{
		"active",
		"inactive",
		"pending",
		"suspended",
		"closed",
	}
	return statuses[rand.IntN(len(statuses))]
}

func businessType() string {
	types := [5]string{
		"retail",
		"service",
		"manufacturing",
		"technology",
		"healthcare",
	}
	return types[rand.IntN(len(types))]
}

// businessAddress returns a full, internally-consistent address rather than
// randomizing street/city/state/zip independently — mixing an unrelated
// street with an unrelated city/state would produce nonsense addresses.
func businessAddress() model.Address {
	addresses := [10]model.Address{
		{Street: "123 Main St", City: "Springfield", State: "IL", Zip: "62704"},
		{Street: "456 Elm St", City: "Austin", State: "TX", Zip: "73301"},
		{Street: "789 Oak St", City: "Denver", State: "CO", Zip: "80202"},
		{Street: "101 Maple Ave", City: "Seattle", State: "WA", Zip: "98101"},
		{Street: "202 Pine St", City: "Boston", State: "MA", Zip: "02110"},
		{Street: "303 Cedar St", City: "Miami", State: "FL", Zip: "33101"},
		{Street: "404 Birch St", City: "Chicago", State: "IL", Zip: "60601"},
		{Street: "505 Walnut St", City: "Phoenix", State: "AZ", Zip: "85001"},
		{Street: "606 Cherry St", City: "Atlanta", State: "GA", Zip: "30301"},
		{Street: "707 Spruce St", City: "Portland", State: "OR", Zip: "97201"},
	}
	return addresses[rand.IntN(len(addresses))]
}

func phoneNumber() string {
	numbers := [10]string{
		"555-555-1234",
		"555-555-5678",
		"555-555-9012",
		"555-555-3456",
		"555-555-7890",
		"555-555-2345",
		"555-555-6789",
		"555-555-0123",
		"555-555-4567",
		"555-555-8901",
	}
	return numbers[rand.IntN(len(numbers))]
}

func randDaySetOfDays() []string {
	days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	selectedDays := make([]string, 0)
	for _, day := range days {
		if rand.IntN(2) == 1 { // 50% chance to include the day
			selectedDays = append(selectedDays, day)
		}
	}

	if len(selectedDays) == 0 {
		selectedDays = append(selectedDays, days[rand.IntN(len(days))]) // Ensure at least one day is selected
	}

	return selectedDays
}

func randTime() string {
	hour := rand.IntN(24)
	minute := rand.IntN(60)
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

// randOpeningHours pairs each open day from randDaySetOfDays with a random
// open/close time. Close isn't guaranteed to fall after Open — fine for
// seed data meant to exercise the API and index, not to be realistic.
func randOpeningHours() []model.OpeningHours {
	days := randDaySetOfDays()
	hours := make([]model.OpeningHours, 0, len(days))
	for _, day := range days {
		hours = append(hours, model.OpeningHours{
			Day:   day,
			Open:  randTime(),
			Close: randTime(),
		})
	}
	return hours
}

func randLatLong() (float64, float64) {
	lat := rand.Float64()*2 + 39
	lon := rand.Float64()*2 - 91
	return lat, lon
}

func randRating() float64 {
	return 1 + rand.Float64()*4 // [1.0, 5.0)
}

func randSquareFootage() int {
	return 500 + rand.IntN(9500) // [500, 9999]
}

func randOpeningDate() time.Time {
	daysAgo := rand.IntN(20 * 365) // sometime in roughly the last 20 years
	return time.Now().AddDate(0, 0, -daysAgo)
}

// randomBusiness assembles one fake Business by calling each field
// generator above.
func randomBusiness() model.Business {
	name := businessName()
	lat, lon := randLatLong()

	return model.Business{
		Name:           name,
		DisplayName:    name,
		BusinessStatus: businessStatus(),
		PrimaryType:    businessType(),
		Address:        businessAddress(),
		Location:       model.GeoPoint{Lat: lat, Lon: lon},
		PhoneNumber:    phoneNumber(),
		SquareFootage:  randSquareFootage(),
		Rating:         randRating(),
		OpeningDate:    randOpeningDate(),
		OpeningHours:   randOpeningHours(),
	}
}

const seedCount = 20

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	es, err := store.NewElasticsearchStore(cfg.ElasticsearchURL, cfg.ElasticsearchAPIKey)
	if err != nil {
		log.Fatalf("connecting to elasticsearch: %v", err)
	}

	ctx := context.Background()
	for i := 0; i < seedCount; i++ {
		business := randomBusiness()

		created, err := es.CreateBusiness(ctx, business)
		if err != nil {
			log.Fatalf("seeding business %d: %v", i, err)
		}

		fmt.Printf("created %q (%s)\n", created.Name, created.ID)
	}
}
