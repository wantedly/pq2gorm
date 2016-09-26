package models

import "time"

type PostComment struct {
	ID        uint       `json:"id"`
	UserID    uint       `json:"user_id"`
	PostID    uint       `json:"post_id"`
	Content   string     `json:"content"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}
