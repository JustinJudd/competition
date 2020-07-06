package storm

import (
	"fmt"
	"sort"

	"github.com/justinjudd/competition/models"
	"github.com/justinjudd/competition/models/storm/pb"

	"github.com/asdine/storm"
	"github.com/asdine/storm/codec/protobuf"
	"github.com/asdine/storm/q"
)

type engine struct {
	*storm.DB
}

// NewStorageEngine creates and returns a StorageEngine meeting the engine interface, using a storm db backend
func NewStorageEngine(path string) (models.StorageEngine, error) {
	db, err := storm.Open(path, storm.Codec(protobuf.Codec))
	//db, err := storm.Open(path) // Use this for debug or if you want JSON stored in the database
	if err != nil {
		return nil, fmt.Errorf("Unable to open storage engine: %w", err)
	}

	return &engine{db}, nil

}

type competition struct {
	pb.Competition
	*storm.DB
}

type tournament struct {
	pb.Tournament
	*storm.DB
}

type arena struct {
	pb.Arena
	*storm.DB
}

type player struct {
	pb.Player
	*storm.DB
}

type team struct {
	pb.Team
	*storm.DB
}

type round struct {
	pb.Round
	*storm.DB
}

type game struct {
	pb.Game
	*storm.DB
}

func (e *engine) CreateCompetition(name string, players []models.Player) models.Competition {
	c := competition{DB: e.DB}
	c.Name = name
	err := e.Save(&c.Competition)
	if err != nil {
		fmt.Println("Unable to create competition:", err)
	}

	return &c
}

func (e *engine) CreatePlayer(name string, metadata []byte) models.Player {
	p := pb.Player{Name: name, Metadata: metadata}
	e.Save(&p)
	return &player{p, e.DB}
}

func (e *engine) GetCompetitions() []models.Competition {
	var comps []models.Competition
	e.Select().Each(new(pb.Competition), func(record interface{}) error {
		c := record.(*pb.Competition)
		comps = append(comps, &competition{*c, e.DB})
		return nil
	})
	return comps
}

func (e *engine) GetPlayers() []models.Player {
	var players []models.Player
	e.Select().Each(new(pb.Player), func(record interface{}) error {
		p := record.(*pb.Player)
		players = append(players, &player{*p, e.DB})
		return nil
	})
	return players
}

func (e *engine) GetPlayer(name string) models.Player {

	var p pb.Player
	err := e.One("Name", name, &p)
	if err != nil {
		fmt.Println("Error getting matching player:", err)
		return nil
	}
	return &player{p, e.DB}
}

func (c *competition) AddTournament(name string, tournamentType models.TournamentType, teams []models.Team, seeded bool, gameSize uint32, advancing uint32, scored bool) models.Tournament {
	//t := tournament{DB: c.DB}
	t := pb.Tournament{Name: name, Type: pb.TournamentType(tournamentType), CompetitionId: c.GetId(), Seeded: seeded, GameSize: gameSize, Advancing: advancing, Scored: scored}
	err := c.Save(&t)
	if err != nil {
		fmt.Println("Error saving tournament:", err)
	}
	tourney := tournament{t, c.DB}
	for _, team := range teams {
		tourney.CreateTeam(team.GetName(), team.GetPlayers(), team.GetMetadata())
	}
	return &tourney
}

func (c *competition) GetActiveTournament() models.Tournament {

	var t []pb.Tournament
	err := c.Find("CompetitionId", c.Id, &t, storm.Limit(1), storm.Reverse())
	if err != nil {
		fmt.Println("Error getting active tournament:", err)
		return nil
	}
	active := t[0]

	return &tournament{active, c.DB}
}

func (c *competition) GetAllTournaments() []models.Tournament {
	var tournies []models.Tournament
	err := c.Select(q.Eq("CompetitionId", c.Id)).Each(new(pb.Tournament), func(record interface{}) error {
		t := record.(*pb.Tournament)
		tournies = append(tournies, &tournament{*t, c.DB})
		return nil
	})
	if err != nil {

	}
	return tournies
}

func (c *competition) GetArenas() []models.Arena {
	var pbArenas []pb.Arena
	err := c.All(&pbArenas)
	if err != nil {

	}
	arenas := make([]models.Arena, len(pbArenas))
	for i, a := range pbArenas {
		arenas[i] = &arena{a, c.DB}
	}
	return arenas
}

func (c *competition) CreateArena(name string) models.Arena {
	a := pb.Arena{Name: name}
	c.Save(&a)
	return &arena{a, c.DB}
}

func (a *arena) GetGames() []models.Game {
	var games []pb.Game
	a.Select(q.Eq("ArenaId", a.Id), q.In("Status", []pb.Status{pb.Status_NEW, pb.Status_ONGOING})).Find(&games)

	outGames := make([]models.Game, len(games))
	for i, g := range games {
		outGames[i] = &game{g, a.DB}
	}

	return outGames
}

func (p *player) SetMetadata(metadata []byte) {
	p.Metadata = metadata
	p.UpdateField(&p.Player, "Metadata", metadata)
}

func (p *player) GetRecords() []models.Game {
	var teamIds []uint64
	p.Select(q.Eq("PlayerId", p.Id)).Each(new(pb.PlayerTeam), func(record interface{}) error {
		pt := record.(*pb.PlayerTeam)
		teamIds = append(teamIds, pt.TeamId)
		return nil
	})
	var gameIds []uint64
	var games []models.Game
	p.Select(q.In("TeamId", teamIds)).Each(new(pb.GameTeam), func(record interface{}) error {
		gt := record.(*pb.GameTeam)
		gameIds = append(gameIds, gt.GameId)
		return nil
	})
	p.Select(q.In("Id", gameIds)).Each(new(pb.Game), func(record interface{}) error {
		g := record.(*pb.Game)
		games = append(games, &game{*g, p.DB})
		return nil
	})
	return games
}

func (t *tournament) NextRound() (models.Round, error) {
	r := pb.Round{Status: pb.Status_NEW, TournamentId: t.Id}
	err := t.Save(&r)
	if err != nil {
		fmt.Println("Error starting a new round:", err)
	}
	return &round{r, t.DB}, err
}

func (t *tournament) getActiveRound() (*pb.Round, error) {
	var round pb.Round

	err := t.Select(q.Eq("TournamentId", t.Id)).Reverse().First(&round)
	if err != nil {
		//r := pb.Round{TournamentId: t.Id}
		//return &r, nil
		return nil, err
	}

	return &round, nil
}

func (t *tournament) GetActiveRound() models.Round {
	r, err := t.getActiveRound()
	if err != nil {
		return nil
	}
	return &round{*r, t.DB}
}

func (t *tournament) StartRound() {
	r, err := t.getActiveRound()
	if err != nil {
		fmt.Println("Error getting active round:", err)
		r = &pb.Round{TournamentId: t.Id}
	}
	r.Status = pb.Status_ONGOING

	err = t.Update(&r)
	if err != nil {
		fmt.Println("Error starting round:", err)
	}
}

func (t *tournament) GetAllRounds() []models.Round {
	var rounds []models.Round
	t.Select(q.Eq("TournamentId", t.Id)).Each(new(pb.Round), func(record interface{}) error {
		r := record.(*pb.Round)
		rounds = append(rounds, &round{*r, t.DB})
		return nil
	})

	return rounds
}

func (t *tournament) GetType() models.TournamentType {
	return models.TournamentType(t.Type)
}

func (t *tournament) SetMetadata(data []byte) {
	t.Metadata = data
	t.UpdateField(&t.Tournament, "Metadata", data)
}

func (t *tournament) GetBracketOrder() []string {
	return nil
}

func (t *tournament) GetTeams() []models.Team {
	var teams []models.Team
	t.Select(q.Eq("TournamentId", t.Id)).Each(new(pb.Team), func(record interface{}) error {
		t1 := record.(*pb.Team)
		teams = append(teams, &team{*t1, t.DB})
		return nil
	})

	return teams
}

func (t *tournament) GetTeam(name string) models.Team {
	var tm pb.Team
	t.Select(q.Eq("TournamentId", t.Id), q.Eq("Name", name)).First(&tm)
	return &team{tm, t.DB}
}

func (t *tournament) IsScored() bool {
	return t.Scored
}

func (t *tournament) IsSeeded() bool {
	return t.Seeded
}

func (t *tournament) SetStatus(status models.Status) {
	t.Status = pb.Status(status)
	t.UpdateField(&t.Tournament, "Status", pb.Status(status))
}

func (t *tournament) SetFinal() {
	// TODO: Mark active rounds and games as completed
	t.Status = pb.Status_COMPLETED
	t.UpdateField(&t.Tournament, "Status", pb.Status_COMPLETED)
}

func (t *tournament) CreateTeam(name string, players []models.Player, metadata []byte) models.Team {
	tm := pb.Team{Name: name, TournamentId: t.Id, Metadata: metadata}
	t.Save(&tm)
	// Map all of the players to this new team
	for _, p := range players {
		var pbPlayer pb.Player
		err := t.One("Name", p.GetName(), &pbPlayer)
		if err != nil {

		}
		pt := pb.PlayerTeam{PlayerId: pbPlayer.Id, TeamId: tm.Id}
		t.Save(&pt)
	}

	return &team{tm, t.DB}
}

func (t *tournament) GetStatus() models.Status {
	return models.Status(t.Status)
}

func (r *round) CreateGame(teams []models.Team, scored bool) models.Game {
	g := pb.Game{RoundId: r.Id, Status: pb.Status_NEW}
	r.Save(&g)

	for _, t := range teams {
		if models.IsByeTeam(t) {
			continue
		}
		var pbTeam pb.Team
		err := r.Select(q.Eq("Name", t.GetName()), q.Eq("TournamentId", r.GetTournamentId())).First(&pbTeam)
		if err != nil {
			fmt.Println("Error assigning teams for this game:", g)
		}
		if t.Equals(&team{pbTeam, r.DB}) {
			gt := pb.GameTeam{GameId: g.Id, TeamId: pbTeam.Id}
			r.Save(&gt)
		}
	}

	return &game{g, r.DB}
}

func (r *round) GetGames() []models.Game {
	var games []models.Game
	err := r.Select(q.Eq("RoundId", r.Id)).Each(new(pb.Game), func(record interface{}) error {
		g := record.(*pb.Game)
		games = append(games, &game{*g, r.DB})
		return nil
	})
	if err != nil {
		fmt.Println("Error getting all games for this round:", err)
	}

	return games
}

func (r *round) SetFinal() {
	r.Status = pb.Status_COMPLETED
	r.UpdateField(&r.Round, "Status", pb.Status_COMPLETED)
}

func (r *round) Start() {
	r.Status = pb.Status_ONGOING
	r.UpdateField(&r.Round, "Status", pb.Status_ONGOING)
}

func (r *round) GetStatus() models.Status {
	return models.Status(r.Status)
}

func (r *round) SetStatus(status models.Status) {
	r.Status = pb.Status(status)
	r.UpdateField(&r.Round, "Status", pb.Status(status))
}

func (g *game) GetTeams() []models.Team {
	var teams []models.Team
	g.Select(q.Eq("GameId", g.Id)).Each(new(pb.GameTeam), func(record interface{}) error {
		gt := record.(*pb.GameTeam)
		var t pb.Team
		// Grab them all in order
		g.Select(q.Eq("Id", gt.TeamId)).First(&t)
		teams = append(teams, &team{t, g.DB})
		return nil
	})

	return teams
}

func (g *game) GetArena() models.Arena {
	var a pb.Arena
	g.One("Id", g.ArenaId, &a)

	return &arena{a, g.DB}
}

func (g *game) SetArena(a models.Arena) {
	var pbArena pb.Arena
	g.One("Name", a.GetName(), &pbArena)
	g.ArenaId = pbArena.Id
	g.UpdateField(&g.Game, "ArenaId", pbArena.Id)
}

func (g *game) SetScores(scores []int64) {
	var gts []pb.GameTeam
	g.Select(q.Eq("GameId", g.Id)).Find(&gts)
	if len(scores) != len(gts) {
		// Throw error
	}

	for i, gt := range gts {
		gt.Score = scores[i]
		g.UpdateField(&gt, "Score", scores[i])
	}

}

func (g *game) SetPlaces(places []int64) {
	var gts []pb.GameTeam
	g.Select(q.Eq("GameId", g.Id)).Find(&gts)
	if len(places) != len(gts) {
		// Throw error
		return
	}
	for i, gt := range gts {
		gt.Place = places[i]
		err := g.UpdateField(&gt, "Place", places[i])
		if err != nil {
			fmt.Println("Unable to update place field:", err)
		}

	}
}

func (g *game) GetScores() []int64 {
	var scores []int64
	g.Select(q.Eq("GameId", g.Id)).Each(new(pb.GameTeam), func(record interface{}) error {
		gt := record.(*pb.GameTeam)
		scores = append(scores, gt.Score)
		return nil
	})

	return scores
}

func (g *game) GetPlaces() []int64 {
	var places []int64
	g.Select(q.Eq("GameId", g.Id)).Each(new(pb.GameTeam), func(record interface{}) error {
		gt := record.(*pb.GameTeam)
		places = append(places, gt.Place)
		return nil
	})

	return places
}

func (g *game) Start() {
	g.Status = pb.Status_ONGOING
	g.UpdateField(g.Game, "Status", pb.Status_ONGOING)
}

func (g *game) SetStatus(status models.Status) {
	g.Status = pb.Status(status)
	err := g.UpdateField(&g.Game, "Status", pb.Status(status))
	if err != nil {
		fmt.Println("Error updating game status:", err)
	}
}

func (g *game) GetStatus() models.Status {
	return models.Status(g.Status)
}

func (g *game) IsScored() bool {
	var r round
	g.One("Id", g.RoundId, &r.Round)
	var t tournament
	g.One("Id", r.TournamentId, &t.Tournament)
	return t.Scored
}

func (g *game) SetBracket(bracket string) {
	g.Bracket = bracket
	err := g.UpdateField(&g.Game, "Bracket", bracket)
	if err != nil {
		fmt.Println("Unable to set bracket:", err)
	}
	return
}

type teamScore struct {
	teamId uint64
	score  int64
	place  int64
}

type teamScores []teamScore

func (t teamScores) Len() int           { return len(t) }
func (t teamScores) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t teamScores) Less(i, j int) bool { return t[i].score < t[j].score }

func (g *game) SetFinal() {
	g.Status = pb.Status_COMPLETED
	g.UpdateField(&g.Game, "Status", pb.Status_COMPLETED)

	// If the game is scored, update places
	if g.IsScored() {
		scores := g.GetScores()
		var gts []pb.GameTeam
		g.Select(q.Eq("GameId", g.Id)).Find(&gts)
		if len(scores) != len(gts) {
			// Throw error
		}

		allScores := teamScores{}
		places := make([]int64, len(scores))
		scoresMap := map[int64]int{} // Map scores to how many teams who scored that score
		teamIndex := map[uint64]int{}

		for i, gt := range gts {
			gt.Score = scores[i]
			scoresMap[scores[i]]++
			allScores = append(allScores, teamScore{gt.TeamId, gt.Score, 0})
			teamIndex[gt.TeamId] = i
		}

		sort.Sort(sort.Reverse(allScores))

		lastPlace := -1
		for _, place := range allScores {
			if scoresMap[place.score] > 1 {
				// There was a tie
				p := lastPlace
				if lastPlace < 0 {
					p *= -1
				}
				place.place = int64(p * -1)
			} else {
				lastPlace++
				place.place = int64(lastPlace)
			}
			places[teamIndex[place.teamId]] = place.place
		}

		for i, gt := range gts {
			gt.Place = places[i]
			g.UpdateField(&gt, "Place", places[i])
		}
	}

	return
}

func (g *game) GetTeamPlace(t models.Team) int64 {
	tActual, ok := t.(*team)
	if !ok {
		return 0
	}
	var gt pb.GameTeam
	err := g.Select(q.Eq("GameId", g.Id), q.Eq("TeamId", tActual.Id)).First(&gt)
	if err != nil {
		return 0
	}

	return gt.GetPlace()
}

func (g *game) GetTeamScore(t models.Team) int64 {
	tActual, ok := t.(*team)
	if !ok {
		return 0
	}
	var gt pb.GameTeam
	err := g.Select(q.Eq("GameId", g.Id), q.Eq("TeamId", tActual.Id)).First(&gt)
	if err != nil {
		return 0
	}

	return gt.GetScore()
}

func (t *team) Equals(t2 models.Team) bool {
	t2Actual, ok := t2.(*team)
	if !ok {
		return false
	}
	return t.Id == t2Actual.Id
}

func (t *team) GetPlayers() []models.Player {
	var playerIds []uint64
	t.Select(q.Eq("TeamId", t.Id)).Each(new(pb.PlayerTeam), func(record interface{}) error {
		pt := record.(*pb.PlayerTeam)
		playerIds = append(playerIds, pt.PlayerId)
		return nil
	})
	var players []models.Player
	t.Select(q.In("Id", playerIds)).Each(new(pb.Player), func(record interface{}) error {
		p := record.(*pb.Player)
		players = append(players, &player{*p, t.DB})
		return nil
	})
	return players
}

func (t *team) GetRecords() []models.Game {
	var gameIds []uint64
	var games []models.Game
	t.Select(q.Eq("TeamId", t.Id)).Each(new(pb.GameTeam), func(record interface{}) error {
		gt := record.(*pb.GameTeam)
		gameIds = append(gameIds, gt.GameId)
		return nil
	})
	t.Select(q.In("Id", gameIds)).Each(new(pb.Game), func(record interface{}) error {
		g := record.(*pb.Game)
		games = append(games, &game{*g, t.DB})
		return nil
	})

	return games
}

func (t *team) IsBye() bool {
	return len(t.GetPlayers()) == 0
}

func (t *team) SetMetadata(data []byte) {
	t.Metadata = data
	t.UpdateField(&t.Team, "Metadata", data)
}

func CreateByeTeam() models.Team {
	return &team{}
}
