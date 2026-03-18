package repository

import (
	"GlitchRoyale/internal/domain"
	"errors"
)

type MemoryGameRepository struct {
	games map[string]*domain.Game
}

func NewMemoryGameRepository() *MemoryGameRepository {
	return &MemoryGameRepository{
		games: make(map[string]*domain.Game),
	}
}

func (r *MemoryGameRepository) Save(game *domain.Game) error {
	r.games[game.ID] = game
	return nil
}

func (r *MemoryGameRepository) GetByID(id string) (*domain.Game, error) {
	game, ok := r.games[id]
	if !ok {
		return nil, errors.New("game not found")
	}
	return game, nil
}
