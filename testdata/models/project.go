package models

type Project struct {
	ID          uint    `json:"id"`
	CompanyID   uint    `json:"company_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	Latitude    float32 `json:"latitude"`
	Longtitude  float32 `json:"longtitude"`
}
