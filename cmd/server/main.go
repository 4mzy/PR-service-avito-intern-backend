package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"pr-reviewer-service/internal/handler"
	"pr-reviewer-service/internal/repository"
	"pr-reviewer-service/internal/router"
	"pr-reviewer-service/internal/service"

	_ "github.com/lib/pq"
)

func main() {
	dbConnStr := os.Getenv("DATABASE_URL")
	if dbConnStr == "" {
		dbConnStr = "postgres://postgres:postgres@localhost:5432/pr_reviewer?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connection established")

	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db, userRepo)
	prRepo := repository.NewPullRequestRepository(db)

	teamService := service.NewTeamService(teamRepo)
	userService := service.NewUserService(userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo, teamRepo)
	statsService := service.NewStatsService(userRepo)
	deactivationService := service.NewDeactivationService(userRepo, prRepo, prService)

	h := handler.NewHandler(teamService, userService, prService, statsService, deactivationService)
	r := router.NewRouter(h)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

