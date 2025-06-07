package app

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type App struct {
	DB      *sql.DB
	JWTKey  []byte
	Schemas map[string]string
}

func Initialize() *App {
	if err := godotenv.Load(); err != nil {
		log.Fatal("error loading .env file: ", err)
	}

	db := SetupDB()
	schemas := loadSchemas()
	jwtKey := []byte(os.Getenv("JWT_SECRET"))

	return &App{
		DB:      db,
		JWTKey:  jwtKey,
		Schemas: schemas,
	}

}

func SetupDB() *sql.DB {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DBHOST"),
		os.Getenv("DBPORT"),
		os.Getenv("DBUSER"),
		os.Getenv("DBPASSWORD"),
		os.Getenv("DBNAME"),
	)

	db, err := sql.Open("postgres", connStr)

	if err != nil {
		log.Fatal("Database connection error:", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Minute * 5)

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	log.Println("Connected to PostgreSQL!")
	return db
}
