package models

import "time"

type PostComments struct {
	ID        uint       `gorm:"primary_key;AUTO_INCREMENT" json:"id"`
	UserID    uint       `json:"user_id"`
	User      *User      `json:"user"` // This line is infered from column name "user_id".
	PostID    uint       `json:"post_id"`
	Post      *Post      `json:"post"` // This line is infered from column name "post_id".
	Content   string     `json:"content"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}
