package domain

type Game struct {
	ID      string
	Players map[string]*Player
	Round   int
	State   string
}

func NewGame(id string) *Game {
	return &Game{
		ID:      id,
		Players: make(map[string]*Player),
		State:   "waiting",
	}
}

func (g *Game) AddPlayer(p *Player) {
	g.Players[p.ID] = p
}

func (g *Game) IsGameOver() bool {
	alive := 0
	for _, p := range g.Players {
		if p.Alive {
			alive++
		}
	}
	return alive <= 1
}
