package accounts

// This file holds the server-owned plan catalog (like tutor.Curriculum, it is
// server data clients render, never decide). Sample pricing only — the exact
// free/paid boundary and prices need sign-off before real checkout
// (Blueprint/monetization_funnel.md §6); real payment is deliberately not wired
// yet. A guardian subscription covers every child they have consented for.

// Plan is one purchasable subscription tier.
type Plan struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PriceNaira  int    `json:"priceNaira"`
	Period      string `json:"period"`
	Description string `json:"description"`
}

// Plans is the sample catalog a guardian can choose from. Edit pricing here
// (single source of truth) before go-live.
func Plans() []Plan {
	return []Plan{
		{
			ID:          "family-monthly",
			Name:        "SoTeach Family — Monthly",
			PriceNaira:  2000,
			Period:      "month",
			Description: "Full tutoring on every subject and topic, for all the children you give consent for.",
		},
		{
			ID:          "family-annual",
			Name:        "SoTeach Family — Annual",
			PriceNaira:  20000,
			Period:      "year",
			Description: "A full year of tutoring for every consented child — two months free on the monthly price.",
		},
	}
}

// planByID validates plan lookups against the sample catalog.
var planByID = func() map[string]Plan {
	m := map[string]Plan{}
	for _, p := range Plans() {
		m[p.ID] = p
	}
	return m
}()
