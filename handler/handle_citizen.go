package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"ivr_ataljanseva/db/repository"
	"ivr_ataljanseva/models"
)

type CitizenHandler struct {
	repo         *repository.CitizenRepository
	politicalRepo *repository.PoliticalUserRepository
}

func NewCitizenHandler(
	repo *repository.CitizenRepository,
	politicalRepo *repository.PoliticalUserRepository,
) *CitizenHandler {
	return &CitizenHandler{
		repo:         repo,
		politicalRepo: politicalRepo,
	}
}

func (h *CitizenHandler) GetCitizen(c *gin.Context) {
	phone := c.Param("phone")

	citizen, err := h.repo.FindByPhone(
		c.Request.Context(),
		phone,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if citizen == nil {
		c.JSON(http.StatusOK, models.CitizenLookupResponse{
			Found: false,
		})
		return
	}

	nagarsevakName := ""

	if citizen.NagarsevakID != uuid.Nil {
		ns, err := h.politicalRepo.FindNagarsevakByID(
			c.Request.Context(),
			citizen.NagarsevakID,
		)
		if err == nil && ns != nil {
			nagarsevakName = ns.Name
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"found":           true,
		"language":        citizen.Language,
		"pincode":         citizen.Pincode,
		"ward":            citizen.Ward,
		"nagarsevak_id":   citizen.NagarsevakID.String(),
		"nagarsevak_name": nagarsevakName,
	})
}


func (h *CitizenHandler) RegisterCitizen(c *gin.Context) {

	var req models.RegisterCitizenRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	err := h.repo.Create(
		c.Request.Context(),
		&req,
	)


	if err != nil {
		log.Printf("register citizen failed: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "citizen registered successfully",
	})
}