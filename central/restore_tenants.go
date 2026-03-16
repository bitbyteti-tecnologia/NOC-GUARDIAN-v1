package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dsn := "postgres://guardian:guardian_strong_password@127.0.0.1:5432/guardian_master?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	tenants := []struct {
		ID     string
		Name   string
		DBName string
	}{
		{
			ID:     "c5f25c4b-23e1-4f03-a519-58ed96c84fb6",
			Name:   "BitByteTI",
			DBName: "tenant_bitbyteti_c5f25c4b",
		},
		{
			ID:     "3f8d7636-b8fa-4934-8e4f-b7952861f694",
			Name:   "Espaco Neuronave",
			DBName: "tenant_espaco_neuronave_3f8d7636",
		},
		{
			ID:     "e1a51cdb-8212-4adb-a5b4-168e7c5af4f8",
			Name:   "NOC Guardian Server",
			DBName: "tenant_noc_guardian_server_e1a51cdb",
		},
	}

	for _, t := range tenants {
		uid, err := uuid.Parse(t.ID)
		if err != nil {
			log.Printf("Error parsing UUID %s: %v", t.ID, err)
			continue
		}

		_, err = pool.Exec(context.Background(),
			"INSERT INTO tenants (id, name, db_name, created_at) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET name = $2, db_name = $3",
			uid, t.Name, t.DBName, time.Now())
		if err != nil {
			log.Printf("Error inserting tenant %s: %v", t.Name, err)
		} else {
			log.Printf("Tenant %s restored successfully", t.Name)
		}
	}
}
