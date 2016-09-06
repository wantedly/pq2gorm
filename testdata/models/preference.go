package models

import "time"

type Preference struct {
	ID                 uint       `json:"id"`
	UserID             uint       `json:"user_id"`
	User               *User      `json:"user"` // This line is infered from column name "user_id"
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	Locale             string     `json:"locale" sql:"DEFAULT:'ja'::character varying"`
	DeletedAt          *time.Time `json:"deleted_at"`
	EmailSubscriptions string     `json:"email_subscriptions"`
	Searchable         bool       `json:"searchable" sql:"DEFAULT:true"`
}
