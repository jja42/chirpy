package main

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func replaceProfaneWord(word string) string {
	profaneWords := [3]string{"kerfuffle", "sharbert", "fornax"}
	word_to_check := strings.ToLower(word)
	for _, prprofaneWord := range profaneWords {
		if word_to_check == prprofaneWord {
			return "****"
		}
	}
	return word
}

func replaceProfaneText(text string) string {
	words := strings.Split(text, " ")
	new_text := make([]string, 0)
	for _, word := range words {
		new_word := replaceProfaneWord(word)
		new_text = append(new_text, new_word)
	}
	cleaned_text := strings.Join(new_text, " ")
	return cleaned_text
}

type UserResponse struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
}

type ChirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}
