package models

// Status is the basic status for tournaments, rounds, and games
type Status int32

const (
	Status_NEW       Status = 0
	Status_ONGOING   Status = 1
	Status_COMPLETED Status = 2
)

// Tournament type is used to discern between different competition types. Both Elimination and Regular season type competitions are supported
type TournamentType int32

const (
	TournamentType_SINGLE_ELIMINATION TournamentType = 0
	TournamentType_DOUBLE_ELIMINATION TournamentType = 1
	TournamentType_ROUND_ROBIN        TournamentType = 2
	TournamentType_COMPASS_DRAW       TournamentType = 3
	TournamentType_SWISS_FORMAT       TournamentType = 4
	TournamentType_GROUP_PLAY         TournamentType = 5
)

// StorageEngine is a backing that provides storing details for an active competition
type StorageEngine interface {
	CreateCompetition(name string, players []Player) Competition
	CreatePlayer(name string, metadata []byte) Player

	GetCompetitions() []Competition
	GetPlayers() []Player
	GetPlayer(name string) Player
}

// Competition is the broadest category here. It can contain multiple tournaments
// As an example, a competition could consist of a regular round robin season followed by a single elimination tournament
type Competition interface {
	AddTournament(name string, tournamentType TournamentType, teams []Team, seeded bool, gameSize uint32, advancing uint32, scored bool) Tournament
	GetActiveTournament() Tournament
	CreateArena(name string) Arena
	GetAllTournaments() []Tournament
	GetArenas() []Arena
	GetName() string
}

// Tournament provides a single competitive type of event, all rankings/matches are of the same type within the tournament
type Tournament interface {
	NextRound() (Round, error)
	StartRound()
	GetActiveRound() Round
	GetAllRounds() []Round
	GetName() string
	GetType() TournamentType
	SetMetadata([]byte)        //store match times in here
	GetMetadata() []byte       //store match times in here
	GetBracketOrder() []string // Get Display/importance order of brackets
	GetTeams() []Team
	IsScored() bool
	CreateTeam(name string, players []Player, metadata []byte) Team //If no players are provided, the team will be used as a BYE team
	GetGameSize() uint32
	IsSeeded() bool
	GetAdvancing() uint32
	SetStatus(Status)
	GetStatus() Status
	SetFinal()
	GetTeam(name string) Team
}

// Round is a single round within a tournament
type Round interface {
	CreateGame(teams []Team, scored bool) Game
	GetGames() []Game
	SetFinal() // Round is over, set all games to final/complete & lock in whatever scores/places are in place
	Start()
	SetStatus(Status)
	GetStatus() Status
}

// Game is a single competitive event
type Game interface {
	GetTeams() []Team
	GetStatus() Status
	SetStatus(Status)
	SetScores([]int64) //map of teamIds to scores // Or should I map team names to scores?
	SetPlaces([]int64) //map of teamIds to places (Only should be called if game is not scored)
	SetFinal()         // Game is over, lock in whatever scores/places are in place
	GetArena() Arena
	SetArena(Arena)
	Start()
	GetBracket() string
	SetBracket(string)
	IsScored() bool
	GetScores() []int64
	GetPlaces() []int64

	GetTeamPlace(t Team) int64
	GetTeamScore(t Team) int64
}

// Team is a participant in a Game, that is part of a competition
type Team interface {
	GetPlayers() []Player
	GetName() string
	SetMetadata([]byte)  //store images in here
	GetMetadata() []byte // Store images in here
	GetRecords() []Game
	Equals(Team) bool
}

// Player is part of a competition, and can be on teams that participate in competitions
type Player interface {
	GetName() string
	SetMetadata([]byte)  //store images in here
	GetMetadata() []byte // Store Images in here
	GetRecords() []Game
}

// Arena is a place for the events to be held at
type Arena interface {
	GetName() string
	GetGames() []Game
}
