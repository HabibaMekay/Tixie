package models

type Vendor struct {
    ID        int    `json:"id"`
    VendorName string `json:"vendor_name"`  
    Email     string `json:"email"`
    Password  string `json:"password"` 
}
