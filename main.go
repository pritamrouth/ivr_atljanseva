package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"ivr_ataljanseva/db"
	"ivr_ataljanseva/db/repository"
	"ivr_ataljanseva/handler"
	"ivr_ataljanseva/routes"
)

func main() {

	database := db.Connect()

	if err := db.AutoMigrate(); err != nil {
		log.Fatal(err)
	}

	baseURL := os.Getenv("PLIVO_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	audioBaseURL := os.Getenv("AUDIO_BASE_URL")
	if audioBaseURL == "" {
		audioBaseURL = baseURL + "/audio"
	}

	router := gin.Default()

	router.Static("/audio", "./audio")

	citizenRepo := repository.NewCitizenRepository(database)
	politicalRepo := repository.NewPoliticalUserRepository(database)

	citizenHandler := handler.NewCitizenHandler(citizenRepo, politicalRepo)
	wardHandler := handler.NewWardHandler(politicalRepo, citizenRepo)
	nagarsevakHandler := handler.NewNagarsevakHandler(politicalRepo, citizenRepo)
	plivoHandler := handler.NewPlivoHandler(citizenRepo, politicalRepo, baseURL, audioBaseURL)

	routes.RegisterRoutes(
		router,
		citizenHandler,
		wardHandler,
		nagarsevakHandler,
		plivoHandler,
	)

	log.Println("server running on :5020")
	log.Fatal(router.Run(":5020"))
}
