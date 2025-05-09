package repos

import (
	"database/sql"
	"errors"
	"vendor-service/internal/db/models"

	circuitbreaker "tixie.local/common"

	"golang.org/x/crypto/bcrypt"
)

type VendorRepository struct {
	DB      *sql.DB
	breaker *circuitbreaker.Breaker
}

func NewVendorRepository(db *sql.DB) *VendorRepository {
	return &VendorRepository{
		DB:      db,
		breaker: circuitbreaker.NewBreaker("vendor-repository"),
	}
}

func (r *VendorRepository) CreateVendor(vendor models.Vendor) error {
	result := r.breaker.Execute(func() (interface{}, error) {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(vendor.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		query := `INSERT INTO vendors (vendor_name, email, password) VALUES ($1, $2, $3)`
		_, err = r.DB.Exec(query, vendor.VendorName, vendor.Email, string(hashedPassword))
		return nil, err
	})
	return result.Error
}

func (r *VendorRepository) GetAllVendors() ([]models.Vendor, error) {
	var vendors []models.Vendor
	result := r.breaker.Execute(func() (interface{}, error) {
		query := `SELECT id, vendor_name, email, password FROM vendors`
		rows, err := r.DB.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var vendor models.Vendor
			err := rows.Scan(&vendor.ID, &vendor.VendorName, &vendor.Email, &vendor.Password)
			if err != nil {
				return nil, err
			}
			vendors = append(vendors, vendor)
		}
		return vendors, nil
	})

	if result.Error != nil {
		return nil, result.Error
	}
	return result.Data.([]models.Vendor), nil
}

func (r *VendorRepository) GetVendorByID(id int) (models.Vendor, error) {
	var vendor models.Vendor
	result := r.breaker.Execute(func() (interface{}, error) {
		query := `SELECT id, vendor_name, email, password FROM vendors WHERE id = $1`
		err := r.DB.QueryRow(query, id).Scan(&vendor.ID, &vendor.VendorName, &vendor.Email, &vendor.Password)
		if err != nil {
			if err == sql.ErrNoRows {
				return vendor, errors.New("vendor not found")
			}
			return vendor, err
		}
		return vendor, nil
	})

	if result.Error != nil {
		return vendor, result.Error
	}
	return result.Data.(models.Vendor), nil
}

func (r *VendorRepository) UpdateVendor(id int, updatedVendor models.Vendor) error {
	result := r.breaker.Execute(func() (interface{}, error) {
		var password string
		if updatedVendor.Password != "" {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updatedVendor.Password), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			password = string(hashedPassword)
		} else {
			password = updatedVendor.Password
		}

		query := `UPDATE vendors SET vendor_name = $1, email = $2, password = $3 WHERE id = $4`
		_, err := r.DB.Exec(query, updatedVendor.VendorName, updatedVendor.Email, password, id)
		return nil, err
	})
	return result.Error
}

func (r *VendorRepository) DeleteVendor(id int) error {
	result := r.breaker.Execute(func() (interface{}, error) {
		query := `DELETE FROM vendors WHERE id = $1`
		_, err := r.DB.Exec(query, id)
		return nil, err
	})
	return result.Error
}

func (r *VendorRepository) CheckCredentials(vendorName, password string) (bool, error) {
	result := r.breaker.Execute(func() (interface{}, error) {
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
	})

	if result.Error != nil {
		return false, result.Error
	}
	return result.Data.(bool), nil
}
