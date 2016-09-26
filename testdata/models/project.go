package models

type Project struct {
	ID          uint     `json:"id"`
	CompanyID   uint     `json:"company_id"`
	Company     *Company `json:"company"` // This line is infered from column name "company_id".
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Location    string   `json:"location"`
	Latitude    float32  `json:"latitude"`
	Longitude   float32  `json:"longitude"`
}
