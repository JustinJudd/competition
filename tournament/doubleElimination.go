package tournament

import (
	"fmt"
	"math"
	"sort"

	"github.com/justinjudd/competition/models"
)

type roundType int

var doubleEliminationBrackets = []string{"Winning Bracket", "Losing Bracket", "Finals"}

const (
	first      roundType = iota // The first round of double elimination
	lMajor                      // Play the loser bracket
	lMinor                      // Wait to fill in the loser's bracket
	noL                         // No teams in the Losers bracket queue
	final                       // Play the final round
	finalExtra                  // Play an extra final round (The team from the losers bracket won, so now each team has one loss)
)

// DoubleElimination fulfills the Tournament interface. Provides the logic for running a Tournament of a Double Elimination type. Commonly used as a conclusion of a season or competition
type DoubleElimination struct {
	models.Tournament
	gameCounter  int
	playIn       int
	gamesBracket map[uint32]int
	roundType
	winnerQue []models.Team
	losersQue []models.Team
}

func NewDoubleElimination(baseTournament models.Tournament) models.Tournament {
	teams := baseTournament.GetTeams()

	tmp := math.Log2(float64(len(teams)))
	var playInGames int
	if tmp != math.Floor(tmp) {
		playInGames = len(teams) - int(math.Pow(math.Floor(tmp), 2.0))
	}
	return &DoubleElimination{baseTournament, 0, playInGames, map[uint32]int{}, 0, []models.Team{}, []models.Team{}}
}

func (d *DoubleElimination) GetBracketOrder() []string {
	return doubleEliminationBrackets
}

func (d *DoubleElimination) GetActiveStage() models.Tournament {
	return d
}

func (d *DoubleElimination) Start() {
	d.SetStatus(models.Status_ONGOING)
}

func (c *DoubleElimination) StartRound() {
	if c.GetStatus() == models.Status_COMPLETED {
		return
	}
	lastRound := c.Tournament.GetActiveRound()
	lastRound.SetStatus(models.Status_ONGOING)
}

func getTeamCountToInclude(teamCount int, gameSize uint32) int {
	tmp := math.Log2(float64(teamCount))
	return int(math.Pow(float64(gameSize), math.Floor(tmp)))
}

func (c *DoubleElimination) NextRound() (models.Round, error) {
	lastRound := c.GetActiveRound()

	var winningTeams, losingTeams []models.Team
	gameSize := int(c.Tournament.GetGameSize())
	moveForward := c.GetAdvancing()

	if len(c.GetAllRounds()) == 0 || lastRound == nil {
		//Create first round
		winningTeams = c.GetTeams()

		idealTeamNum := int(math.Pow(2.0, math.Ceil(math.Log2(float64(len(winningTeams))))))
		for i := len(winningTeams); i < idealTeamNum; i++ {
			winningTeams = append(winningTeams, nil)
		}
		if c.IsSeeded() {
			winningTeams = seed(winningTeams)

		}
	} else {

		if c.roundType == first {
			c.roundType = lMajor
		}

		if lastRound.GetStatus() != models.Status_COMPLETED {
			return nil, fmt.Errorf("Can't start new round until previous round is completed")
		}

		losingQue := []models.Team{}
		for _, game := range lastRound.GetGames() {
			teams := game.GetTeams()
			places := game.GetPlaces()

			teamSlice := make([]TeamScore, 0)
			for i, teamPlaced := range places {
				if len(teams) <= i {
					break
				}
				if models.IsByeTeam(teams[i]) {
					continue
				}
				teamSlice = append(teamSlice, TeamScore{teams[i], int(teamPlaced)})
			}

			sort.Slice(teamSlice, BasicTeamScoreLess(teamSlice))

			switch game.GetBracket() {
			case doubleEliminationBrackets[0]: //Winner's Bracket
				for _, teamPlaced := range teamSlice[:moveForward] { // Winners stay in the Winner's Bracket
					c.winnerQue = append(c.winnerQue, teamPlaced.Team)
				}
				for _, teamPlaced := range teamSlice[moveForward:] { // Losers go to the Loser's Bracket
					losingQue = append(losingQue, teamPlaced.Team)
				}
			case doubleEliminationBrackets[1]: // Losers Bracket
				for _, teamPlaced := range teamSlice[:moveForward] { // Only Winners stay in, losers are eliminated
					c.losersQue = append(c.losersQue, teamPlaced.Team)
				}
			case doubleEliminationBrackets[2]: // Final round(s)
				winner := teamSlice[0].Team
				records := winner.GetRecords()
				prevGame := records[len(records)-1]
				bracket := prevGame.GetBracket()
				switch bracket {
				case doubleEliminationBrackets[0]: // Winner of the Winner's Bracket won the final round (only one game needed)
					fmt.Println("GAME OVER", winner.GetName(), "has won")
					c.SetStatus(models.Status_COMPLETED)
					return nil, fmt.Errorf("Too many rounds @ %d", len(c.GetAllRounds()))
				case doubleEliminationBrackets[1]: // Winner of the Loser's Bracket one, second final round will be needed
					// Need second final round
					c.roundType = finalExtra
					fmt.Println("Need second final round")
					for _, teamPlaced := range teamSlice[:moveForward] {
						c.winnerQue = append(c.winnerQue, teamPlaced.Team)
					}
					for _, teamPlaced := range teamSlice[moveForward:] {
						losingQue = append(losingQue, teamPlaced.Team)
					}
				case doubleEliminationBrackets[2]: // The second Final Round was played, winner of it is the overall winner
					fmt.Println("GAME OVER", winner.GetName(), "has won")
					c.SetStatus(models.Status_COMPLETED)
					return nil, fmt.Errorf("Too many rounds @ %d", len(c.GetAllRounds()))

				}
			}

		}

		c.losersQue = append(c.losersQue, losingQue...)

		rounds := c.GetAllRounds()

		if len(rounds) < 1 {

			switch len(rounds) {
			case 0:
				winningTeams = c.winnerQue
				c.winnerQue = []models.Team{}
			case 1: // Handle losers play-in game
				losersSize := len(c.losersQue)
				for i := 0; i < c.playIn; i++ {
					losingTeams = append(losingTeams, c.losersQue[i])
					losingTeams = append(losingTeams, c.losersQue[losersSize-1-i])
				}
				c.losersQue = c.losersQue[c.playIn:]
				c.losersQue = c.losersQue[:losersSize-1-c.playIn]
			}
		} else {
			switch c.roundType {
			case first, noL, lMinor:
				c.roundType = lMajor
			case lMajor:
				c.roundType = lMinor
			}
			if len(c.losersQue) == 0 {
				c.roundType = noL
			}

			if len(c.losersQue) > gameSize/2 {
				losingTeams = c.losersQue
				c.losersQue = []models.Team{}
			} else {
				// How many teams should I pull?
			}

			gameSize := c.GetGameSize()
			if len(c.winnerQue) > int(c.GetAdvancing()) {
				winningTeams = c.winnerQue
				c.winnerQue = []models.Team{}
			}

			// Figure out if we need to move to the Final round
			if (len(winningTeams) == 0 && len(losingTeams) == 0) &&
				((len(c.winnerQue) <= int(c.GetAdvancing()) && len(c.losersQue) <= int(c.GetAdvancing())) ||
					(len(c.winnerQue) == 0 && len(c.losersQue) == int(gameSize))) {

				c.roundType = final
			}

		}
	}

	r, err := c.Tournament.NextRound()
	if err != nil {
		return r, err
	}
	r.SetStatus(models.Status_NEW)

	if c.roundType == final {
		teams := []models.Team{}
		teams = append(teams, c.winnerQue...)
		teams = append(teams, c.losersQue...)
		c.winnerQue = []models.Team{}
		c.losersQue = []models.Team{}
		game := r.CreateGame(teams, c.IsScored())
		game.SetBracket(doubleEliminationBrackets[2])

	} else {
		half := gameSize / 2
		if len(winningTeams) > int(c.GetAdvancing()) {
			//avoidRematches(winningTeams, int(gameSize))
			gameCount := int(math.Ceil(float64(len(winningTeams)) / float64(gameSize)))
			shortGames := gameCount*gameSize - len(winningTeams)
			if len(winningTeams) < gameSize {
				shortGames = 1
			}
			for i := 0; i < gameCount-shortGames; i++ {
				game := r.CreateGame(winningTeams[i*gameSize:(i+1)*gameSize], c.IsScored())
				game.SetBracket(doubleEliminationBrackets[0])

			}
			place := (gameCount - shortGames) * gameSize
			for i := gameCount - shortGames; i < (gameCount - 1); i++ {
				game := r.CreateGame(winningTeams[place:place+gameSize-1], c.IsScored())
				game.SetBracket(doubleEliminationBrackets[0])

			}
			if shortGames > 0 {
				game := r.CreateGame(winningTeams[place:], c.IsScored())
				game.SetBracket(doubleEliminationBrackets[0])
			}

		}

		if c.roundType == lMinor { //after first round, we should "shuffle" the teams up so that the people that just dropped from the winning round aren't playing each other
			if len(losingTeams) > 0 && len(c.GetAllRounds()) != 2 {
				halfTeamPoint := len(losingTeams) / 2
				for i, team := range losingTeams[:halfTeamPoint] {
					if i%2 == 1 { //"shuffle" every other
						continue
					}
					team2 := losingTeams[i+halfTeamPoint]
					losingTeams[i], losingTeams[i+halfTeamPoint] = team2, team
				}
			}
		}

		if len(losingTeams) > half { // Check if there are enough teams to play the losers bracket
			//avoidRematches(losingTeams, int(gameSize))
			gameCount := int(math.Ceil(float64(len(losingTeams)) / float64(gameSize)))
			shortGames := gameCount*gameSize - len(losingTeams)
			for i := 0; i < gameCount-shortGames; i++ {
				game := r.CreateGame(losingTeams[i*gameSize:(i+1)*gameSize], c.IsScored())
				game.SetBracket(doubleEliminationBrackets[1])
			}
			place := (gameCount - shortGames) * gameSize
			for i := gameCount - shortGames; i < gameCount; i++ {
				shortTeams := gameSize - 1
				if i < 0 {
					shortTeams--
					i++
				}
				game := r.CreateGame(losingTeams[place:place+shortTeams], c.IsScored())
				game.SetBracket(doubleEliminationBrackets[1])
				place += shortTeams
			}
		}

	}

	return r, nil
}
