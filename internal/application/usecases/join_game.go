package usecases

import (
	"GlitchRoyale/internal/application/ports"
	"GlitchRoyale/internal/domain"
)

type JoinGame struct {
	repo ports.GameRepository
}

func NewJoinGame(repo ports.GameRepository) *JoinGame {
	return &JoinGame{repo: repo}
}

func (uc *JoinGame) Execute(gameID, playerID string) (*domain.Game, error) {
	game, err := uc.repo.GetByID(gameID)
	if err != nil {
		return nil, err
	}

	player := domain.NewPlayer(playerID)
	game.AddPlayer(player)

	err = uc.repo.Save(game)
	return game, err
}
