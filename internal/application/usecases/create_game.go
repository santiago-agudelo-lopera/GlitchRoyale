package usecases

import (
	"GlitchRoyale/internal/application/ports"
	"GlitchRoyale/internal/domain"
)

type CreateGame struct {
	repo ports.GameRepository
}

func NewCreateGame(repo ports.GameRepository) *CreateGame {
	return &CreateGame{repo: repo}
}

func (uc *CreateGame) Execute(id string) (*domain.Game, error) {
	game := domain.NewGame(id)
	err := uc.repo.Save(game)
	return game, err
}
