package ports

import "GlitchRoyale/internal/domain"

type GameRepository interface {
	Save(game *domain.Game) error
	GetByID(id string) (*domain.Game, error)
}
