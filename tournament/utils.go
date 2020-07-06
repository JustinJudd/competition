package tournament

import (
	"math"

	"github.com/justinjudd/competition/models"
)

func orderPivots(n int) []int {
	ordered := []int{0}
	for i := n - 1; i > 0; i-- {
		ordered = append(ordered, i)
	}

	return ordered
}

type Pivot struct {
	Place int
	Span  int
}

func seed(teams []models.Team) []models.Team {

	teamSize := int(math.Pow(2.0, math.Ceil(math.Log2(float64(len(teams))))))
	for i := len(teams); i < teamSize; i++ {
		teams = append(teams, nil)
	}
	orderedTeams := make([]models.Team, teamSize)
	count := 0
	pivots := []Pivot{{0, len(teams)}}
	for {
		if count >= len(teams)/2 {
			break
		}
		ordered := orderPivots(len(pivots))
		order := 0
		for _, i := range ordered {
			pivot := pivots[i]
			p := pivot.Place
			p2 := p + 1
			if order%2 != 0 {
				p = p - 1
				if p < 0 {
					p += pivot.Span
				}
				p2 = p - 1
			}

			orderedTeams[p] = teams[count]
			orderedTeams[p2] = teams[len(teams)-(count+1)]
			count++
			order++
		}
		for i := len(ordered) - 1; i >= 0; i-- {
			pivot := pivots[ordered[i]]
			p := pivot.Place
			p2 := p + 1
			if order%2 != 0 {
				p = p - 1
				if p < 0 {
					p += pivot.Span
				}
				p2 = p - 1
			}

			orderedTeams[p] = teams[count]
			orderedTeams[p2] = teams[len(teams)-(count+1)]
			count++
			order++
		}

		newPivots := []Pivot{}
		for _, pivot := range pivots {
			span := pivot.Span / 2
			p := pivot.Place - span
			if p > 0 && p < len(teams) {
				newPivots = append(newPivots, Pivot{p, span})
			}
			p = pivot.Place + span
			if p > 0 && p < len(teams) {
				newPivots = append(newPivots, Pivot{p, span})
			}
		}
		pivots = newPivots

	}

	return orderedTeams

}

type TeamScore struct {
	Team  models.Team
	Score int
}

func avoidRematches(teams []models.Team, groupSize int) [][]models.Team {
	type edge struct {
		From, To int
	}
	teamIndexes := map[string]int{}
	for i, team := range teams {
		teamIndexes[team.GetName()] = i
	}
	costMap := map[edge]int{}
	for aIndex, team := range teams {
		records := team.GetRecords()
		for i, record := range records {
			for _, teamB := range record.GetTeams() {
				if !team.Equals(teamB) {
					costMap[edge{aIndex, teamIndexes[teamB.GetName()]}] = i + 1
				}

			}
		}
	}

	minCost := 100
	var minTeams [][]int

	base := make([]int, len(teams))
	for i := 0; i < len(teams); i++ {
		base[i] = i
	}

	permShifts := make([]int, len(base))
	var workingTeam [][]int

	gameCount := int(math.Ceil(float64(len(teams)) / float64(groupSize)))
	shortGames := gameCount*groupSize - len(teams)
	splits := make([]int, gameCount)
	index := 0
	for i := 0; i < gameCount-shortGames; i++ {
		index += groupSize
		splits[i] = index
	}
	for i := gameCount - shortGames; i < gameCount; i++ {
		index += gameCount - 1
		splits[i] = index
	}
	if splits[gameCount-1] > len(teams) { // This should be an error, the last splits should be the same as the number of teams

	}

	breakPoint := gameCount
	if groupSize < gameCount {
		breakPoint = groupSize
	}
	for permShifts[0] < len(permShifts) {

		if permShifts[0] < breakPoint && minCost <= gameCount {
			break
		}
		working := make([]int, len(base))
		copy(working, base)
		for i, v := range permShifts {
			working[i], working[i+v] = working[i+v], working[i]
		}

		workingTeam = [][]int{}
		for j := 0; j < gameCount; j++ {
			start := 0
			end := splits[j]
			if j != 0 {
				start = splits[j-1]
			}
			workingTeam = append(workingTeam, working[start:end])
		}
		cost := 0
		for _, group := range workingTeam {
			for i, a := range group {
				for j, b := range group {
					if i == j {
						continue
					}

					cost += costMap[edge{a, b}]
				}
			}
		}

		if cost < minCost {
			minTeams = workingTeam
			minCost = cost
		}
		if cost == 0 {
			break
		}

		for i := len(permShifts) - 1; i >= 0; i-- {
			if i == 0 || permShifts[i] < len(permShifts)-i-1 {
				permShifts[i]++
				break
			}
			permShifts[i] = 0
		}
	}

	preparedTeams := [][]models.Team{}
	for _, group := range minTeams {
		game := []models.Team{}
		for _, team := range group {
			game = append(game, teams[team])
		}
		preparedTeams = append(preparedTeams, game)
	}

	//fmt.Println(minCost, minTeams)

	return preparedTeams
}

func BasicTeamScoreLess(teams []TeamScore) func(i, j int) bool {

	return func(i, j int) bool {
		t1, t2 := teams[i], teams[j]
		return FlipTies(t1.Score) < FlipTies(t2.Score)
	}
}

func FlipTies(place int) int {
	if place >= 0 {
		return place
	}
	return (place - 1) * -1
}
