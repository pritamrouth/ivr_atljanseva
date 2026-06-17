package routes

import (
	"github.com/gin-gonic/gin"

	"ivr_ataljanseva/handler"
)

func RegisterRoutes(
	router *gin.Engine,
	citizenHandler *handler.CitizenHandler,
	wardHandler *handler.WardHandler,
	nagarsevakHandler *handler.NagarsevakHandler,
) {
	ivr := router.Group("/ivr")
	{
		ivr.GET("/citizen/:phone", citizenHandler.GetCitizen)
		ivr.POST(
			"/register/citizen",
			citizenHandler.RegisterCitizen,
		)
		ivr.POST(
			"/register/resolve",
			wardHandler.ResolveWard,
		)
		ivr.POST(
			"/nagarsevak",
			nagarsevakHandler.FindNagarsevak,
		)
		ivr.POST(
			"/citizen/complete",
			nagarsevakHandler.CompleteCitizen,
		)
	}
}