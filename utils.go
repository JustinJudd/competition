package competition

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"math/rand"
	"reflect"
	"strings"

	"github.com/justinjudd/competition/models"
)

func GenerateTournamentHTML(t models.Tournament) ([]byte, error) {
	bracketNames := t.GetBracketOrder()
	rounds := t.GetAllRounds()

	var out []byte

	out = append(out, []byte("<h1>"+t.GetName()+"</h1>")...)

	brackets := map[string]Bracket{}

	for i, round := range rounds {
		//fmt.Println("Setting up bracket for round:", round)
		for _, b := range bracketNames {
			//brackets[b].Rounds[i] = make([]models.Game, 0)
			bracket := brackets[b]
			bracket.Name = b
			bracket.Advance = t.GetAdvancing()
			bracket.Rounds = append(bracket.Rounds, make([]models.Game, 0))
			bracket.Scored = t.IsScored()
			bracket.FinalWinner = t.GetType() != models.TournamentType_ROUND_ROBIN && t.GetType() != models.TournamentType_GROUP_PLAY
			brackets[b] = bracket
		}
		//fmt.Println(brackets)
		for _, game := range round.GetGames() {
			//fmt.Println("Getting bracket for:", game)
			b := game.GetBracket()
			bracket := brackets[b]
			bracket.Advance = t.GetAdvancing()
			bracket.Rounds[i] = append(bracket.Rounds[i], game)
		}

	}

	//fmt.Println(brackets)

	for _, b := range bracketNames {
		h, err := brackets[b].FancyHTML()
		if err != nil {
			return nil, err
		}
		out = append(out, h...)

	}

	return out, nil
}

type CompetitionOverError error

type Table struct {
	Name     string
	Gamesize int
	Rows     [][]models.Team
	RowWidth []int
}

const tableHTML = `
<table>
<caption>{{.Name}}</caption>
{{ range $i, $row := .Rows }}
    <tr>
    {{ range $j, $team := $row -}}
        {{$drawTop := drawTopBar $i $j}}
        {{$drawBottom := drawBottomBar $i $j}}
        <td rowspan="{{ colWidth $j }}" style="border-left: solid 1px black; border-right: solid 1px black;{{if $drawBottom}} border-bottom: solid 1px black;{{end}}{{if $drawTop}} border-top: solid 1px black;{{end}}">{{if $team}}{{$team.Name}}{{else}}BYE{{end}}</td>
    {{ end -}}</tr>
{{ end }}</table>
`

func (t *Table) ToHTML() ([]byte, error) {
	funcMap := template.FuncMap{
		"colWidth": func(n int) int {
			return t.RowWidth[n]
		},
		"drawBottomBar": func(row, round int) bool {

			if row == len(t.Rows)-1 {
				return true
			}
			if row+t.RowWidth[round] == len(t.Rows) {
				return true
			}

			//return false
			// Bottom Bar
			if (row+t.RowWidth[round])%(t.RowWidth[round]*t.Gamesize) == 0 {
				return true
			}

			return false
		},
		"drawTopBar": func(row, round int) bool {
			if row == 0 {
				return true
			}
			if row%(t.RowWidth[round]*t.Gamesize) == 0 {
				return true
			}

			return false
		},
	}
	tmpl, err := template.New("table").Funcs(funcMap).Parse(tableHTML)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, t)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

const bracketHTML = `
{{- $scored := .Scored -}}
<h4>{{.Name}}</h4>
<main class="bracket">
{{$winner := lastWinner}}
{{ range $i, $round := .Rounds }}
    <ul>
    {{ range $j, $game := $round -}}
        
        {{ $showGame := showGame $game -}}
        
		{{ range $k, $team := $game.GetTeams -}}
			{{ if $team }}
				{{ if $showGame }}
				<li class="game{{if eq $k 0}} game-top{{end}}{{if last $k $game.GetTeams }} game-bottom{{end}}{{if winner $game $team }} winner{{end}}">{{if $team.Metadata}}<img src="{{printf "%s" $team.Metadata}}">{{else}}<span></span>{{end}}{{$team.Name}} <span>{{$notBye := not $team.IsBye}}{{if and $scored $notBye }}{{score $game $team}}{{end}}</span></li>
				{{ else }}
				<li class="game{{if eq $k 0}} game-top{{end}}{{if last $k $game.GetTeams }} game-bottom{{end}}">  <span></span></li>
				{{ end }}
			{{ end -}}
            
        {{end -}}
        {{if last $j $round | not }}<li>&nbsp;</li> {{end}}
    {{ end -}}</ul>
{{ end }}{{if $winner}}<ul><li class="game round-winner">{{if $winner.Metadata}}<img src="{{printf "%s" $winner.Metadata}}">{{else}}<span></span>{{end}}{{$winner.Name}} <span></span></li></ul>{{end}}
</main>
`

type Bracket struct {
	Name        string
	Rounds      [][]models.Game //[]models.Round
	Scored      bool
	Advance     uint32
	FinalWinner bool
}

func (b Bracket) FancyHTML() ([]byte, error) {

	funcMap := template.FuncMap{
		"last": func(x int, a interface{}) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
		"winner": func(game models.Game, team models.Team) bool {
			return IsWinner(team, game, b.Advance)
		},
		"score": func(game models.Game, team models.Team) int {
			var located int
			for i, t := range game.GetTeams() {
				if team.Equals(t) {
					located = i
					break
				}
			}
			return int(game.GetScores()[located])

		},
		"lastWinner": func() models.Team {
			if len(b.Rounds) == 0 {
				return nil
			}
			lastRound := b.Rounds[len(b.Rounds)-1]
			if len(lastRound) == 0 {
				return nil
			}

			lastMatch := lastRound[0]
			if lastMatch.GetStatus() != models.Status_COMPLETED {
				return nil
			}
			if !b.FinalWinner {
				return nil
			}
			for _, team := range lastMatch.GetTeams() {
				if IsWinner(team, lastMatch, b.Advance) {
					return team
				}
			}

			return nil

		},
		"showGame": func(g models.Game) bool {
			return true
		},
	}
	tmpl, err := template.New("bracket").Funcs(funcMap).Parse(bracketHTML)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, b)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil

}

const gameHTML = `
{{- $scored := .scored -}}
{{ $game := .game }}
{{ $completed := complete $game}}
<div class="mini-bracket" style="min-height:100px">
{{if complete $game}}
<div style="text-align:center"><b>Final</b></div>
{{else}}
{{if $game.Arena}}
<div style="text-align:center"><a href="{{arenaURL}}"><b>{{$game.Arena.Name}}</b></a></div>
{{end}}
{{end}}
    <ul>
        {{ range $k, $team := $game.GetTeams -}}
		{{ $place := index $game.Places $k}}
            <li class="game{{if eq $k 0}} game-top{{end}}{{if last $k $game.getTeams }} game-bottom{{end}}{{if winner $game $team }} winner{{end}} {{if and $place (not $completed) }}placed{{end}}">{{if $team.Metadata}}<img src="{{$team.Metadata}}">{{else}}<span></span>{{end}}{{$team.Name}} <span>{{$notBye := not $team.IsBye}}{{if and $scored $notBye}}{{score $game $team}}{{end}}</span></li>
        {{end -}}</ul>
</div>`

func GameToHTML(g models.Game, scored bool) ([]byte, error) {

	if g.GetArena().GetName() == "" {
		return nil, nil
	}

	funcMap := template.FuncMap{
		"last": func(x int, a interface{}) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
		"winner": func(game models.Game, team models.Team) bool {
			if team == nil { // team was a bye team
				return false
			}
			if game.GetStatus() != models.Status_COMPLETED {
				return false
			}
			var located int
			for i, t := range game.GetTeams() {
				if t == nil { // bye team
					continue
				}
				if team.Equals(t) {
					located = i
					break
				}
			}
			place := FlipTies(int(game.GetPlaces()[located]))
			return place < int(math.Ceil(float64(len(game.GetTeams()))/2))
		},
		"score": func(game models.Game, team models.Team) int {
			var located int
			for i, t := range game.GetTeams() {
				if team.Equals(t) {
					located = i
					break
				}
			}
			return int(game.GetScores()[located])

		},
		"complete": func(game models.Game) bool {
			return game.GetStatus() == models.Status_COMPLETED
		},
		"arenaURL": func() string {
			return "/arena/" + strings.ToLower(strings.Replace(g.GetArena().GetName(), " ", "", -1))
		},
	}
	tmpl, err := template.New("game").Funcs(funcMap).Parse(gameHTML)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{"game": g, "scored": scored})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

const bigGameHTML = `
{{- $scored := .scored -}}
{{ $game := .game}}
{{ $completed := complete $game}}
<div class="mdl-grid bigGame">
{{range $i, $team := $game.GetTeams }}
	{{ $place := index $game.Places $i}}
  <div class="mdl-cell mdl-cell--{{width}}-col mdl-cell--{{tabletWidth}}-col-tablet mdl-cell--{{mobileWidth}}-col-phone {{backgroundColor $i}}"> <div class="team">
    {{ if $team.Metadata }}<span><img src="{{$team.Metadata}}" /></span>{{end}}
	<h3 class="{{if winner $game $team }}winner{{end}}{{if displayPlace $place }} mdl-badge {{ if not $completed }} placed {{end}} {{end}}" {{ if displayPlace $place }}data-badge="{{ displayPlace $place }}"{{end}}>{{$team.Name}}</h3>
	{{if $scored}}<h3 {{if winner $game $team }}class="winner"{{end}}>{{index $game.Scores $i}}</h3>{{end}}
  </div></div>
{{ end }}
</div>
`

func BigGameToHTML(g models.Game, scored bool) ([]byte, error) {

	if g.GetArena().GetName() == "" {
		return nil, nil
	}

	funcMap := template.FuncMap{
		"last": func(x int, a interface{}) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
		"winner": func(game models.Game, team models.Team) bool {
			if models.IsByeTeam(team) { // team was a bye team
				return false
			}
			if game.GetStatus() != models.Status_COMPLETED {
				return false
			}
			var located int
			for i, t := range game.GetTeams() {
				if models.IsByeTeam(t) { // bye team
					continue
				}
				if t.GetName() == team.GetName() {
					located = i
					break
				}
			}
			place := FlipTies(int(game.GetPlaces()[located]))
			//return place < len(game.Teams)/2
			return place < int(math.Ceil(float64(len(game.GetTeams()))/2))
		},
		"score": func(game models.Game, team models.Team) int {
			var located int
			for i, t := range game.GetTeams() {
				if t.GetName() == team.GetName() {
					located = i
					break
				}
			}
			return int(game.GetScores()[located])

		},
		"complete": func(game models.Game) bool {
			return game.GetStatus() == models.Status_COMPLETED
		},
		"width": func() int {
			return 12 / len(g.GetTeams())
		},
		"tabletWidth": func() int {
			return 8 / len(g.GetTeams())
		},
		"mobileWidth": func() int {
			return 4 / len(g.GetTeams())
		},
		"backgroundColor": func(n int) string {
			if len(g.GetTeams()) == 2 {
				return ""
			}
			switch n {
			case 0:
				return "mdc-bg-blue-200"
			case 1:
				return "mdc-bg-red-200"
			case 2:
				return "mdc-bg-yellow-200"
			case 3:
				return "mdc-bg-green-200"
			}
			return ""
		},
		"displayPlace": func(place int32) int32 {
			if g.GetStatus() == models.Status_COMPLETED {
				place++
			}
			return place
		},
	}
	tmpl, err := template.New("BigGame").Funcs(funcMap).Parse(bigGameHTML)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{"game": g, "scored": scored})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func RandomizeTeams(teams []models.Team) {
	places := rand.Perm(len(teams))
	tmp := make([]models.Team, len(teams))
	copy(tmp, teams)
	for i, place := range places {
		teams[i] = tmp[place]
	}
}

func cleanTables(tables []Table) {
	for i, table := range tables {
		for j := len(table.Rows) - 1; j >= 0; j-- {
			row := table.Rows[j]
			if len(row) == 0 {
				table.Rows = append(table.Rows[:j], table.Rows[j+1:]...)
			}
		}
		tables[i] = table
	}
}

func FlipTies(place int) int {
	if place >= 0 {
		return place
	}
	return (place + 1) * -1
}

func IsWinner(t models.Team, g models.Game, advance uint32) bool {
	if models.IsByeTeam(t) {
		return false
	}
	if g.GetStatus() != models.Status_COMPLETED { // If game isn't finished, we shouldn't have a winner
		return false
	}
	var index int
	for i, team := range g.GetTeams() {
		if team.Equals(t) {
			index = i
		}
	}
	place := FlipTies(int(g.GetPlaces()[index]))
	losing := 0
	for i, p := range g.GetPlaces() {
		if i == index {
			continue
		}
		otherPlace := FlipTies(int(p))
		if place > otherPlace {
			losing++
		}
		if place == otherPlace && index > i {
			losing++
		}
	}
	return losing < int(advance)
}

type TeamScore struct {
	Team  models.Team
	Score int
}

func BasicTeamScoreLess(teams []TeamScore) func(i, j int) bool {

	return func(i, j int) bool {
		t1, t2 := teams[i], teams[j]
		return FlipTies(t1.Score) < FlipTies(t2.Score)
	}
}

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
	fmt.Println("ordered by rank")
	for _, t := range teams {
		fmt.Println(t.GetName())
	}
	teamSize := int(math.Pow(2.0, math.Ceil(math.Log2(float64(len(teams))))))
	for i := len(teams); i < teamSize; i++ {
		//teams = append(teams, nil)
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
	for _, t := range orderedTeams {
		if t == nil {
		}
	}
	return orderedTeams

}

func playRound(r models.Round) {
	for _, game := range r.GetGames() {
		if game.GetStatus() != models.Status_COMPLETED {
			playRandomGame(game)
		}
	}
	r.SetFinal()
}

func playRandomGame(g models.Game) {

	places := make([]int64, len(g.GetTeams()))
	placeCounter := 0
	byeTeams := map[int]bool{}
	for i, team := range g.GetTeams() {
		if models.IsByeTeam(team) {
			places[i] = int64(len(g.GetTeams()))
			byeTeams[i] = true
		}

	}

	places2 := rand.Perm(len(g.GetTeams()) - len(byeTeams))
	for i := range g.GetTeams() {
		if byeTeams[i] {
			continue
		}
		place := places2[placeCounter]
		places[i] = int64(place)
		placeCounter++
	}
	g.SetPlaces(places)

}

func playRoundWithScores(r models.Round, maxScore int, allowTies bool) {
	for _, game := range r.GetGames() {
		playRandomGameWithScores(game, maxScore, allowTies)
	}
	r.SetFinal()
}

func playRandomGameWithScores(g models.Game, maxScore int, allowTies bool) {
	scores := make([]int64, len(g.GetTeams()))
	for i, team := range g.GetTeams() {
		if models.IsByeTeam(team) {
			scores[i] = 0
		} else {
			scores[i] = rand.Int63n(int64(maxScore))
		}
	}
	g.SetScores((scores))
}

func avoidRematches(teams []models.Team, groupSize int) [][]models.Team {
	type edge struct {
		From, To int
	}
	costMap := map[edge]int{}
	for aIndex, team := range teams {
		records := team.GetRecords()
		for i, record := range records {
			for bIndex, teamB := range record.GetTeams() {
				if !team.Equals(teamB) {
					costMap[edge{aIndex, bIndex}] = i + 1
				}

			}
		}

	}

	minCost := 100
	var minTeams [][]int

	totalPermutations := 1
	for i := 1; i <= len(teams); i++ {
		totalPermutations *= i
	}

	indexes := make([]int, len(teams))
	for i := 0; i < len(teams); i++ {
		indexes[i] = i
	}
	var workingTeam [][]int

	gameCount := len(teams) / groupSize
	shortGames := gameCount*groupSize - len(teams)
	splits := make([]int, gameCount)
	index := 0
	for i := 0; i < gameCount-shortGames; i++ {
		index += gameCount
		splits[i] = index
	}
	for i := gameCount - shortGames; i < gameCount; i++ {
		index += gameCount - 1
		splits[i] = index
	}
	if splits[gameCount-1] > len(teams) { // This should be an error, the last splits should be the same as the number of teams

	}
	for i := 0; i <= totalPermutations; i++ {
		workingTeam = [][]int{}
		for j := 0; j < gameCount; j++ {
			start := 0
			end := splits[j]
			if j != 0 {
				start = splits[j-1]
			}
			workingTeam = append(workingTeam, indexes[start:end])
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

		indexes[groupSize-1]++
		if i%len(teams) == 0 {
			for j := groupSize - 1; j >= 1; j-- {
				if indexes[j] == groupSize {
					indexes[j] = 0
					indexes[j-1]++
				}
			}
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
