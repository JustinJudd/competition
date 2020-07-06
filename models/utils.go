package models

// IsByeTeam determines if a team is a bye team. StorageEngines may indicate this by either a nil team, or a team with no players
func IsByeTeam(t Team) bool {
	if t == nil {
		return true
	}
	if len(t.GetPlayers()) == 0 {
		return true
	}
	return false
}

// IsByeGame determines if a game should be a bye, and is determined if more real teams are participating than would advance from the game
func IsByeGame(g Game, advance int) bool {
	if g == nil {
		return true
	}
	teams := g.GetTeams()
	if len(teams) <= advance {
		return true
	}
	realTeamCount := 0
	for _, team := range teams {
		if !IsByeTeam(team) {
			realTeamCount++
		}
	}
	if realTeamCount <= advance {
		return true
	}
	return false
}
