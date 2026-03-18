package domain

type Player struct {
	ID     string
	HP     int
	Tokens int
	Alive  bool
}

func NewPlayer(id string) *Player {
	return &Player{
		ID:     id,
		HP:     100,
		Tokens: 0,
		Alive:  true,
	}
}

func (p *Player) ApplyDamage(damage int) {
	p.HP -= damage
	if p.HP <= 0 {
		p.HP = 0
		p.Alive = false
	}
}

func (p *Player) AddToken() {
	p.Tokens++
}
