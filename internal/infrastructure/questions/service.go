package questions

import (
	"context"
	"database/sql"
	"errors"
)

type Answer struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"is_correct"`
}

type Question struct {
	ID         string   `json:"id"`
	Question   string   `json:"question"`
	Category   string   `json:"category"`
	Difficulty string   `json:"difficulty"`
	Answers    []Answer `json:"answers"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetRandomQuestion(ctx context.Context) (Question, error) {
	const questionQuery = `
		SELECT id, question, category, difficulty
		FROM questions
		ORDER BY RANDOM()
		LIMIT 1
	`

	var question Question
	var category sql.NullString
	var difficulty sql.NullString

	err := s.db.QueryRowContext(ctx, questionQuery).Scan(
		&question.ID,
		&question.Question,
		&category,
		&difficulty,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Question{}, errors.New("no questions available")
		}

		return Question{}, err
	}

	question.Category = category.String
	question.Difficulty = difficulty.String

	const answersQuery = `
		SELECT id, text, is_correct
		FROM answers
		WHERE question_id = $1
		ORDER BY id
	`

	rows, err := s.db.QueryContext(ctx, answersQuery, question.ID)
	if err != nil {
		return Question{}, err
	}
	defer rows.Close()

	answers := make([]Answer, 0, 4)
	seenAnswerIDs := make(map[string]struct{}, 4)

	for rows.Next() {
		var answer Answer
		if err := rows.Scan(&answer.ID, &answer.Text, &answer.IsCorrect); err != nil {
			return Question{}, err
		}

		if _, exists := seenAnswerIDs[answer.ID]; exists {
			continue
		}

		seenAnswerIDs[answer.ID] = struct{}{}
		answers = append(answers, answer)
	}

	if err := rows.Err(); err != nil {
		return Question{}, err
	}

	question.Answers = answers
	return question, nil
}
