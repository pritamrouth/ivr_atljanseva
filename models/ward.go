package models

import "github.com/google/uuid"


type ResolveWardRequest struct {
	Phone     string `json:"phone" binding:"required"`
	Pincode   string `json:"pincode" binding:"required"`
	WardInput string `json:"ward_input" binding:"required"`
	Language  string `json:"language" binding:"required"`
}

type ResolveWardResponse struct {
	Status         string    `json:"status"`
	Ward           string    `json:"ward,omitempty"`
	NagarsevakID   uuid.UUID `json:"nagarsevak_id,omitempty"`
	NagarsevakName string    `json:"nagarsevak_name,omitempty"`
	Wards          []string  `json:"wards,omitempty"`
}

type WardMatch struct {
	Ward           string
	NagarsevakID   uuid.UUID
	NagarsevakName string
	NagarsevakPhone string
}

type NagarsevakResponse struct {
    Status    string             `json:"status"`
    AutoSaved bool               `json:"auto_saved,omitempty"`
    Nagarsevak *NagarsevakItem   `json:"nagarsevak,omitempty"`
    List      []NagarsevakItem   `json:"list,omitempty"`
}

type NagarsevakItem struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type NagarsevakLookupRequest struct {
	PhoneNumber string `json:"phone_number"`
	Language    string `json:"language"`
	Pincode     string `json:"pincode"`
	Ward        string `json:"ward"`
}

type NagarsevakRecord struct {
	ID    uuid.UUID
	Name  string
	Phone string
}