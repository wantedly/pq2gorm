package models

import "time"

type Preference struct {
	ID                 uint       `json:"id"`
	UserID             uint       `json:"user_id"`
	CreatedAt          *time.Time `json:"created_at"`
	UpdatedAt          *time.Time `json:"updated_at"`
	Locale             string     `json:"locale" sql:"DEFAULT:'ja'::character varying"`
	DeletedAt          *time.Time `json:"deleted_at"`
	Birthday           *time.Time `json:"birthday"`
	EmailSubscriptions string     `json:"email_subscriptions"`
	Searchable         bool       `json:"searchable" sql:"DEFAULT:true"`
}
