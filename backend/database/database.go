package database

import (
	"log"

	"brunocollective_inventory/config"
	"brunocollective_inventory/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(cfg *config.Config) {
	var err error
	DB, err = gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	err = DB.AutoMigrate(
		&models.User{},
		&models.Product{},
		&models.Customer{},
		&models.Order{},
		&models.OrderItem{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	seedAdmin()
}

func seedAdmin() {
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		admin := models.User{
			Username: "admin",
			Password: string(hash),
		}
		DB.Create(&admin)
		log.Println("Default admin created — username: admin, password: admin123")
	}
}
