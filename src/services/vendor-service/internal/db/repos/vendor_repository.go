package repos

import (
	"database/sql"
	"errors"
	"vendor-service/internal/db/models"

	"golang.org/x/crypto/bcrypt"
)

type VendorRepository struct {
	DB *sql.DB
}

func NewVendorRepository(db *sql.DB) *VendorRepository {
	return &VendorRepository{DB: db}
}

func (r *VendorRepository) CreateVendor(vendor models.Vendor) error {
	
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(vendor.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	query := `INSERT INTO vendors (vendor_name, email, password) VALUES ($1, $2, $3)`
	_, err = r.DB.Exec(query, vendor.VendorName, vendor.Email, string(hashedPassword))
	return err
}


func (r *VendorRepository) GetAllVendors() ([]models.Vendor, error) {
	query := `SELECT id, vendor_name, email, password FROM vendors`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vendors []models.Vendor
	for rows.Next() {
		var vendor models.Vendor
		err := rows.Scan(&vendor.ID, &vendor.VendorName, &vendor.Email, &vendor.Password)
		if err != nil {
			return nil, err
		}
		vendors = append(vendors, vendor)
	}

	return vendors, nil
}


func (r *VendorRepository) GetVendorByID(id int) (models.Vendor, error) {
	query := `SELECT id, vendor_name, email, password FROM vendors WHERE id = $1`
	var vendor models.Vendor
	err := r.DB.QueryRow(query, id).Scan(&vendor.ID, &vendor.VendorName, &vendor.Email, &vendor.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return vendor, nil
		}
		return vendor, err 
	}
	return vendor, nil
}


func (r *VendorRepository) UpdateVendor(id int, updatedVendor models.Vendor) error {

	var password string
	if updatedVendor.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updatedVendor.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		password = string(hashedPassword)
	} else {
		
		password = updatedVendor.Password
	}

	query := `UPDATE vendors SET vendor_name = $1, email = $2, password = $3 WHERE id = $4`
	_, err := r.DB.Exec(query, updatedVendor.VendorName, updatedVendor.Email, password, id)
	return err
}


func (r *VendorRepository) DeleteVendor(id int) error {
	query := `DELETE FROM vendors WHERE id = $1`
	_, err := r.DB.Exec(query, id)
	return err
}


func (r *VendorRepository) CheckCredentials(vendorName, password string) (bool, error) {
	var storedPassword string

	query := `SELECT password FROM vendors WHERE vendor_name = $1`
	err := r.DB.QueryRow(query, vendorName).Scan(&storedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil 
		}
		return false, err 
	}

	
	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		return false, nil 
	}

	return true, nil 
}

