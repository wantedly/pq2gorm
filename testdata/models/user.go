package models

import "time"

type User struct {
	ID           uint           `json:"id"`
	CreatedAt    *time.Time     `json:"created_at"`
	UpdatedAt    *time.Time     `json:"updated_at"`
	DeletedAt    *time.Time     `json:"deleted_at"`
	Preferences  []*Preference  `json:"preferences"`   // This line is infered from other tables.
	PostComments []*PostComment `json:"post_comments"` // This line is infered from other tables.

}
