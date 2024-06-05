package site

import (
	"math"
	"net/http"
	"slices"
	"strings"

	"github.com/delaneyj/datastar"
	"github.com/delaneyj/toolbelt"
	"github.com/go-chi/chi/v5"
	"github.com/go-faker/faker/v4"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

func setupExamplesActiveSearch(examplesRouter chi.Router) error {

	users := make([]*ActiveSearchUser, 256)
	for i := range users {
		u := &ActiveSearchUser{
			ID:        faker.UUIDHyphenated(),
			FirstName: faker.FirstName(),
			LastName:  faker.LastName(),
			Email:     faker.Email(),
		}
		users[i] = u
	}

	examplesRouter.Get("/active_search/data", func(w http.ResponseWriter, r *http.Request) {
		store := &ActiveSearchStore{}
		if err := datastar.QueryStringUnmarshal(r, store); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		scores := map[string]float64{}
		s := strings.ToLower(store.Search)
		minScore, maxScore := math.MaxFloat64, -math.MaxFloat64
		for _, u := range users {
			fn := strings.ToLower(u.FirstName)
			ln := strings.ToLower(u.LastName)
			e := strings.ToLower(u.Email)

			fnd := fuzzy.LevenshteinDistance(s, fn)
			lnd := fuzzy.LevenshteinDistance(s, ln)
			ed := fuzzy.LevenshteinDistance(s, e)

			dist := float64(fnd + lnd + ed)
			scores[u.ID] = dist

			minScore = min(minScore, dist)
			maxScore = max(maxScore, dist)
		}

		for id, score := range scores {
			scores[id] = toolbelt.Fit(score, minScore, maxScore, 100, 0)
		}

		// copy users
		filteredUsers := make([]*ActiveSearchUser, len(users))
		copy(filteredUsers, users)

		// sort users by score
		slices.SortFunc(filteredUsers, func(a, b *ActiveSearchUser) int {
			return int(10000 * (scores[b.ID] - scores[a.ID]))
		})

		sse := datastar.NewSSE(w, r)
		datastar.RenderFragmentTempl(sse, ActiveSearchComponent(filteredUsers, scores, store))
	})

	return nil
}
