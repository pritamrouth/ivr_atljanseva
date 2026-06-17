package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ivr_ataljanseva/db/repository"
	"ivr_ataljanseva/models"
)

type NagarsevakHandler struct {
	politicalRepo *repository.PoliticalUserRepository
	citizenRepo   *repository.CitizenRepository
}

func NewNagarsevakHandler(
	politicalRepo *repository.PoliticalUserRepository,
	citizenRepo *repository.CitizenRepository,
) *NagarsevakHandler {
	return &NagarsevakHandler{
		politicalRepo: politicalRepo,
		citizenRepo:   citizenRepo,
	}
}

func (h *NagarsevakHandler) FindNagarsevak(c *gin.Context) {
	var req models.NagarsevakLookupRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nagarsevaks, err := h.politicalRepo.FindNagarsevaks(
		c.Request.Context(),
		req.Pincode,
		req.Ward,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	count := len(nagarsevaks)

	switch {
	case count == 0:
		c.JSON(http.StatusOK, models.NagarsevakResponse{
			Status: "not_found",
		})

	case count == 1:
		ns := nagarsevaks[0]

		err := h.citizenRepo.UpsertCitizen(
			c.Request.Context(),
			req.PhoneNumber,
			req.Language,
			req.Pincode,
			req.Ward,
			ns.ID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, models.NagarsevakResponse{
			Status:    "single",
			AutoSaved: true,
			Nagarsevak: &models.NagarsevakItem{
				ID:   ns.ID.String(),
				Name: ns.Name,
			},
		})

	case count >= 2 && count <= 5:
		list := make([]models.NagarsevakItem, 0, count)

		for _, ns := range nagarsevaks {
			list = append(list, models.NagarsevakItem{
				ID:   ns.ID.String(),
				Name: ns.Name,
			})
		}

		c.JSON(http.StatusOK, models.NagarsevakResponse{
			Status: "choose",
			List:   list,
		})

	default:
		c.JSON(http.StatusOK, models.NagarsevakResponse{
			Status: "too_many",
		})
	}
}
