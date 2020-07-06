package tournament

import (
	"fmt"
	"strings"

	"github.com/justinjudd/competition/models"
)

// Group Competition fulfills the Tournament interface. Provides the logic for running something like different leagues or groups within a larger tournament or season
type GroupCompetition struct {
	models.Tournament
	children []models.Tournament
}

// NewGroupCompetition creates and returns a Group Competition tournament wrapping the provided children tournaments, and uses the provided backing StorageEngine tournamnent
func NewGroupCompetition(children []models.Tournament, baseTournament models.Tournament) models.Tournament {
	g := GroupCompetition{baseTournament, children}
	return &g
}

func (g *GroupCompetition) GetBracketOrder() []string {
	brackets := []string{}
	for _, child := range g.children {
		for _, bracket := range child.GetBracketOrder() {
			brackets = append(brackets, child.GetName()+":"+bracket)
		}

	}

	return brackets
}

func (g *GroupCompetition) StartRound() {
	for _, child := range g.children {
		lastRound := child.GetActiveRound()
		lastRound.SetStatus(models.Status_ONGOING)

		lastRound.Start()
	}
}

func (g *GroupCompetition) NextRound() (models.Round, error) {

	var grouped groupRound

	for _, child := range g.children {
		lastRound := child.GetActiveRound()
		if lastRound != nil && lastRound.GetStatus() != models.Status_COMPLETED {
			return nil, fmt.Errorf("Can't start new round until previous round is completed")
		}

		round, err := child.NextRound()

		if err != nil {
			return nil, err
		}

		grouped.rounds = append(grouped.rounds, round)

	}
	return &grouped, nil

}

type groupRound struct {
	rounds []models.Round
}

func (r groupRound) CreateGame(teams []models.Team, scored bool) models.Game { //Don't actually support creating games through this, null operation
	return nil
}

func (r groupRound) GetGames() []models.Game {
	var games []models.Game
	for _, round := range r.rounds {
		games = append(games, round.GetGames()...)
	}
	return games
}

func (r groupRound) SetFinal() {
	for _, round := range r.rounds {
		round.SetFinal()
	}
}

func (r groupRound) Start() {
	for _, round := range r.rounds {
		round.Start()
	}

	r.SetStatus(models.Status_ONGOING)
}

func (r groupRound) SetStatus(status models.Status) {
	for _, round := range r.rounds {
		round.SetStatus(status)
	}
}

func (r groupRound) GetStatus() models.Status { // If there are any rounds, return what status the first one is in, otherwise return that it is new
	for _, round := range r.rounds {
		return round.GetStatus()
	}
	return models.Status_NEW
}

func (g *GroupCompetition) GetRounds() models.Round {
	rounds := groupRound{}
	for _, child := range g.children {
		r := child.GetAllRounds()
		rounds.rounds = append(rounds.rounds, r...)
	}
	return rounds

}

func (g *GroupCompetition) GetActiveRound() models.Round {
	rounds := groupRound{}
	for _, child := range g.children {
		r := child.GetActiveRound()
		rounds.rounds = append(rounds.rounds, r)
	}
	return rounds
}

func (g *GroupCompetition) GetAllRounds() []models.Round {
	allRounds := map[int][]models.Round{}
	rounds := []models.Round{}
	maxRounds := 0
	for _, child := range g.children {
		r := child.GetAllRounds()
		for i, round := range r {
			allRounds[i] = append(allRounds[i], round)
			for _, game := range round.GetGames() {
				if strings.Contains(game.GetBracket(), child.GetName()) {
					continue
				}
				game.SetBracket(child.GetName() + ":" + game.GetBracket())
			}
		}
		if len(r) > maxRounds {
			maxRounds = len(r)
		}
	}

	for i := 0; i < maxRounds; i++ {
		r := groupRound{allRounds[i]}
		rounds = append(rounds, r)
	}

	return rounds

}
