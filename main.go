package main

import (
	"log"

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

	router := gin.Default()

	citizenRepo := repository.NewCitizenRepository(database)
	politicalRepo := repository.NewPoliticalUserRepository(database)

	citizenHandler := handler.NewCitizenHandler(citizenRepo, politicalRepo)
	wardHandler := handler.NewWardHandler(politicalRepo, citizenRepo)
	nagarsevakHandler := handler.NewNagarsevakHandler(politicalRepo, citizenRepo)

	routes.RegisterRoutes(
		router,
		citizenHandler,
		wardHandler,
		nagarsevakHandler,
	)

	log.Println("server running on :8080")
	log.Fatal(router.Run(":8080"))
}