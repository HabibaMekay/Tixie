package models

type Vendor struct {
	ID         int    `json:"id"`
	VendorName string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}
