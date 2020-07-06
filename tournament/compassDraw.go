package tournament

import (
	"fmt"
	"math"
	"math/rand"
	"sort"

	"github.com/justinjudd/competition/models"
)

// CompassDraw fulfills the Tournament interface. Provides the logic for running a Tournament of a Compass Draw type. Commonly used in Tennis
type CompassDraw struct {
	models.Tournament
	gameCounter         int
	gameCount           int
	divisionAssignments map[string]int // Key is the team name
	gameDivisions       map[models.Game]int
}

// NewCompassDraw creates and returns a Compass Draw tournament, using the base tournamnet from a StorageEngine
func NewCompassDraw(baseTournament models.Tournament) models.Tournament {
	teams := baseTournament.GetTeams()
	gameSize := baseTournament.GetGameSize()

	gameCount := int(math.Ceil(float64(len(teams)) / float64(gameSize)))
	assignments := map[string]int{}
	gameAssignments := map[models.Game]int{}
	return &CompassDraw{baseTournament, 0, gameCount, assignments, gameAssignments}
}

var compassDivisions = [][]int{
	{0},
	{0, 8},
	{0, 4, 8, 12},
	{0, 2, 4, 6, 8, 10, 12, 14},
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
}

// CompassDivisionNames will be used for the bracket names
var CompassDivisionNames = []string{"East", "East-northeast", "Northeast", "North-northeast", "North", "North-northwest", "Northwest", "West-northwest", "West", "West-southwest", "Southwest", "South-southwest", "South", "South-southeast", "Southeast", "East-southeast"}

func (c *CompassDraw) Start() {
	c.SetStatus(models.Status_ONGOING)
}

func (c *CompassDraw) GetBracketOrder() []string {
	return CompassDivisionNames
}

func (c *CompassDraw) GetActiveStage() models.Tournament {
	return c
}

func (c *CompassDraw) StartRound() {
	round := c.Tournament.GetActiveRound()
	round.Start()
}

func (c *CompassDraw) NextRound() (models.Round, error) {
	lastRound := c.GetActiveRound()

	var teams = make([][]models.Team, len(CompassDivisionNames))
	gameSize := int(c.Tournament.GetGameSize())
	moveForward := c.Tournament.GetAdvancing()
	rounds := c.GetAllRounds()

	if len(rounds) == 0 || lastRound == nil {
		//Create first round
		teams[0] = c.GetTeams()
	} else {
		if lastRound.GetStatus() != models.Status_COMPLETED {
			return nil, fmt.Errorf("Can't start new round until previous round is completed")
		}

		if len(rounds) >= len(compassDivisions) {
			return nil, fmt.Errorf("All matches played")
		}
		roundCount := len(rounds)

		for _, game := range lastRound.GetGames() {
			gameTeams := game.GetTeams()
			divChange := compassDivisions[roundCount][1]
			teamSlice := make([]TeamScore, len(gameTeams))
			for i, teamPlaced := range game.GetPlaces() {
				teamSlice[i] = TeamScore{gameTeams[i], int(teamPlaced)}
			}
			sort.Slice(teamSlice, BasicTeamScoreLess(teamSlice))

			// Teams that lost need to move down a division
			for _, teamPlaced := range teamSlice[moveForward:] {
				c.divisionAssignments[teamPlaced.Team.GetName()] += divChange
			}

		}
		for _, t := range c.GetTeams() {
			teams[c.divisionAssignments[t.GetName()]] = append(teams[c.divisionAssignments[t.GetName()]], t)
		}
	}

	half := gameSize / 2
	if len(teams[0]) <= half {
		c.SetStatus(models.Status_COMPLETED)
		return nil, fmt.Errorf("Not enough teams for another round")
	}

	r, err := c.Tournament.NextRound()
	if err != nil {
		return r, err
	}
	r.SetStatus(models.Status_NEW)

	insufficientTeams := []models.Team{}
	for _, groupTeams := range teams {
		if len(groupTeams) <= half {
			insufficientTeams = append(insufficientTeams, groupTeams...)
			continue
		}
		//avoidRematches(groupTeams, int(gameSize))
		rand.Shuffle(len(groupTeams), func(i, j int) {
			groupTeams[i], groupTeams[j] = groupTeams[j], groupTeams[i]
		})
		gameCount := int(math.Ceil(float64(len(groupTeams)) / float64(gameSize)))
		shortGames := gameCount*gameSize - len(groupTeams)
		for i := 0; i < gameCount-shortGames; i++ {
			game := r.CreateGame(groupTeams[i*gameSize:(i+1)*gameSize], c.IsScored())
			game.SetBracket(CompassDivisionNames[c.divisionAssignments[groupTeams[i*gameSize].GetName()]])
		}
		place := (gameCount - shortGames) * gameSize
		if shortGames > gameCount {
			place = 0
		}
		for i := gameCount - shortGames; i < gameCount; i++ {
			shortTeams := gameSize - 1
			if i < 0 {
				shortTeams--
				i++
			}
			game := r.CreateGame(groupTeams[place:place+shortTeams], c.IsScored())
			game.SetBracket(CompassDivisionNames[c.divisionAssignments[groupTeams[place].GetName()]])
			place += shortTeams
		}

	}

	return r, nil
}
