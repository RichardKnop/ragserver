package ragservertest

import (
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

func New(seed int64, now time.Time) *DataGen {
	g := DataGen{
		Faker: gofakeit.New(seed),
		now:   now.UTC().Truncate(time.Millisecond),
	}

	return &g
}

type DataGen struct {
	*gofakeit.Faker
	now time.Time
}
