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
	plivoHandler *handler.PlivoHandler,
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

		plivo := ivr.Group("/plivo")
		{
			plivo.POST("/incoming", plivoHandler.Incoming)
			plivo.POST("/language", plivoHandler.LanguageSelect)
			plivo.POST("/ward-input", plivoHandler.WardInput)
			plivo.POST("/ward-select", plivoHandler.WardSelect)
			plivo.POST("/nagarsevak-select", plivoHandler.NagarsevakSelect)
			plivo.POST("/main-menu", plivoHandler.MainMenu)
			plivo.POST("/sos-menu", plivoHandler.SOSMenu)
			plivo.POST("/complaint-record", plivoHandler.ComplaintRecord)
			plivo.POST("/complaint-callback", plivoHandler.ComplaintCallback)
		}
	}
}