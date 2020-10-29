package main

import (
	"testing"
	"time"
)

func TestTimeStartOfYearPlusWeeks(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		t.Error(err)
	}

	var tests = []struct {
		year int
		week int

		expectYear  int
		expectMonth int
		expectDay   int
	}{
		{
			2020, 1,
			2019, 12, 30,
		},
		{
			2020, 2,
			2020, 1, 6,
		},
		{
			2020, 25,
			2020, 6, 15,
		},
		{
			2020, 40,
			2020, 9, 28,
		},
		{
			2020, 46,
			2020, 11, 9,
		},
	}

	for _, test := range tests {
		got := timeStartOfYearPlusWeeks(test.year, loc, test.week)
		expect := time.Date(test.expectYear, time.Month(test.expectMonth), test.expectDay, 0, 0, 0, 0, loc)
		assertTimeEquals(t, expect, got)
	}
}

func assertTimeEquals(t *testing.T, expect, got time.Time) {
	if expect != got {
		t.Error("Expected:", expect, " got:", got)
	}
}
