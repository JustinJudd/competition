package tournament

import (
	"fmt"
	"math"

	"github.com/justinjudd/competition/models"
)

// RoundRobin fulfills the Tournament interface, and provides the logic for tournaments where every team plays every other team
type RoundRobin struct {
	models.Tournament
	totalRounds int
}

// NewRoundRobin creates an returns a new Round Robin Tournamnet that uses the provided base tournament StorageEngine
func NewRoundRobin(baseTournament models.Tournament) models.Tournament {
	return &RoundRobin{baseTournament, 0}
}

func (c *RoundRobin) GetBracketOrder() []string {
	return []string{""}
}

func (c *RoundRobin) Start() {
	// Total rounds is decided by looking at how many other teams a team needs to play, and then how many of those teams can be played against each round.
	// If all games in a round don't have the same amount of teams, then an extra round is needed
	numTeams := len(c.GetTeams())
	gameSize := int(c.GetGameSize())
	totalRounds := int(math.Ceil(float64(numTeams-1) / float64(gameSize-1)))
	if (numTeams/gameSize)*gameSize != numTeams {
		totalRounds++
	}
	c.totalRounds = totalRounds

	c.SetStatus(models.Status_ONGOING)

}

func (c *RoundRobin) StartRound() {

	round := c.Tournament.GetActiveRound()
	round.Start()
}

func (c *RoundRobin) NextRound() (models.Round, error) {

	gameSize := int(c.Tournament.GetGameSize())
	teams := c.GetTeams()

	rounds := c.Tournament.GetAllRounds()

	if len(rounds) == 0 {
		//Create first round
		c.Start()

	} else {
		lastRound := c.Tournament.GetActiveRound()
		if lastRound.GetStatus() != models.Status_COMPLETED {
			return nil, fmt.Errorf("Can't start new round until previous round is completed")
		}

		if len(rounds) >= c.totalRounds {
			c.SetStatus(models.Status_COMPLETED)
			return nil, fmt.Errorf("All matches played")
		}

	}

	teamsSplit := avoidRematches(teams, gameSize)

	r, err := c.Tournament.NextRound()
	if err != nil {
		return r, err
	}

	for _, t := range teamsSplit {
		r.CreateGame(t, c.IsScored())
	}

	return r, nil
}
