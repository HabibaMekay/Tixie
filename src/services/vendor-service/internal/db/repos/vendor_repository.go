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
	breaker *circuitbreaker.CircuitBreaker
}

func NewVendorRepository(db *sql.DB) *VendorRepository {
	return &VendorRepository{
		DB:      db,
		breaker: circuitbreaker.NewCircuitBreaker(circuitbreaker.DefaultSettings("vendor-repository")),
	}
}

func (r *VendorRepository) CreateVendor(vendor models.Vendor) error {
	return r.breaker.Execute(func() error {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(vendor.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		query := `INSERT INTO vendors (vendor_name, email, password) VALUES ($1, $2, $3)`
		_, err = r.DB.Exec(query, vendor.VendorName, vendor.Email, string(hashedPassword))
		return err
	})
}

func (r *VendorRepository) GetAllVendors() ([]models.Vendor, error) {
	var vendors []models.Vendor
	err := r.breaker.Execute(func() error {
		query := `SELECT id, vendor_name, email, password FROM vendors`
		rows, err := r.DB.Query(query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var vendor models.Vendor
			err := rows.Scan(&vendor.ID, &vendor.VendorName, &vendor.Email, &vendor.Password)
			if err != nil {
				return err
			}
			vendors = append(vendors, vendor)
		}
		return nil
	})
	return vendors, err
}

func (r *VendorRepository) GetVendorByID(id int) (models.Vendor, error) {
	var vendor models.Vendor
	err := r.breaker.Execute(func() error {
		query := `SELECT id, vendor_name, email, password FROM vendors WHERE id = $1`
		err := r.DB.QueryRow(query, id).Scan(&vendor.ID, &vendor.VendorName, &vendor.Email, &vendor.Password)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		return nil
	})
	return vendor, err
}

func (r *VendorRepository) UpdateVendor(id int, updatedVendor models.Vendor) error {
	return r.breaker.Execute(func() error {
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
	})
}

func (r *VendorRepository) DeleteVendor(id int) error {
	return r.breaker.Execute(func() error {
		query := `DELETE FROM vendors WHERE id = $1`
		_, err := r.DB.Exec(query, id)
		return err
	})
}

func (r *VendorRepository) CheckCredentials(vendorName, password string) (bool, error) {
	var valid bool
	err := r.breaker.Execute(func() error {
		var storedPassword string
		query := `SELECT password FROM vendors WHERE vendor_name = $1`
		err := r.DB.QueryRow(query, vendorName).Scan(&storedPassword)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				valid = false
				return nil
			}
			return err
		}

		err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
		if err != nil {
			valid = false
			return nil
		}

		valid = true
		return nil
	})
	return valid, err
}
