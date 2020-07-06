package tournament

import (
	"fmt"
	"sort"

	"github.com/justinjudd/competition/models"
)

// SingleElimination fulfills the Tournament interface. Provides the logic for running a Tournament of a Single Elimination type. Commonly used as a conclusion of a season or competition
type SingleElimination struct {
	models.Tournament
}

// NewSingleElimination creates a new Single Elimination Tournament
func NewSingleElimination(name string, teams []models.Team, seeded bool, gameSize uint32, advance uint32, scored bool, baseTournament models.Tournament) models.Tournament {
	return &SingleElimination{baseTournament}
}

func (s *SingleElimination) GetBracketOrder() []string {
	return []string{"Main"}
}

func (s *SingleElimination) GetActiveStage() models.Tournament {
	return s
}

func (s *SingleElimination) Start() {
	s.SetStatus(models.Status_ONGOING)
}

func (s *SingleElimination) StartRound() {
	round := s.Tournament.GetActiveRound()
	round.Start()
}

func (s *SingleElimination) NextRound() (models.Round, error) {

	var teams []models.Team
	gameSize := int(s.Tournament.GetGameSize())
	moveForward := s.Tournament.GetAdvancing()
	rounds := s.GetAllRounds()
	if len(rounds) == 0 {
		//Create first round
		teams = s.GetTeams()
		old := make([]models.Team, len(teams))
		copy(old, teams)
		if s.Tournament.IsSeeded() {
			teams = seed(teams)
		}
		for _, t := range old {
			if t == nil {
				continue
			}
			found := false
			for _, t2 := range teams {
				if t2 == nil {
					continue
				}
				if t.Equals(t2) {
					found = true
					break
				}

			}
			if !found {
				fmt.Println("Team", t.GetName(), "didn't get seeded")
			}
		}
	} else {
		lastRound := s.Tournament.GetActiveRound()
		if lastRound.GetStatus() != models.Status_COMPLETED {
			return nil, fmt.Errorf("Can't start new round until previous round is completed")
		}
		for _, game := range lastRound.GetGames() {
			gameTeams := game.GetTeams()
			teamSlice := make([]TeamScore, len(gameTeams))
			for i, teamPlaced := range game.GetPlaces() {
				teamSlice[i] = TeamScore{gameTeams[i], int(teamPlaced)}
			}
			sort.Slice(teamSlice, BasicTeamScoreLess(teamSlice))
			for _, teamPlaced := range teamSlice[:moveForward] {
				teams = append(teams, teamPlaced.Team)
			}

		}
	}

	if len(teams) < gameSize {
		s.Tournament.SetStatus(models.Status_COMPLETED)
		return nil, fmt.Errorf("Not enough teams for another round")
	}

	if !s.Tournament.IsSeeded() {
		avoidRematches(teams, gameSize)
	}

	r, err := s.Tournament.NextRound()
	if err != nil {
		return r, err
	}
	r.SetStatus(models.Status_NEW)
	for i := 0; i < len(teams)/gameSize; i++ {
		r.CreateGame(teams[i*gameSize:(i+1)*gameSize], s.IsScored())
	}

	return r, nil
}
