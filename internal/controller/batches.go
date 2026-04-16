package controller

import (
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type CreateBatchPayload struct {
	Name string `json:"name" binding:"required"`
}

func GetBatches(c *gin.Context) {
	var batches []model.EventBatch
	if err := config.DB.Order("created_at DESC").Find(&batches).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch batches")
		return
	}
	response.OKWithMessage(c, "Batches fetched successfully", batches)
}

func CreateBatch(c *gin.Context) {
	var payload CreateBatchPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	tx := config.DB.Begin()

	// Deactivate all existing batches
	if err := tx.Model(&model.EventBatch{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "Failed to deactivate existing batches")
		return
	}

	// Create new batch as active
	newBatch := model.EventBatch{
		Name:     payload.Name,
		IsActive: true,
	}
	if err := tx.Create(&newBatch).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "Failed to create batch")
		return
	}

	tx.Commit()
	response.CreatedWithMessage(c, "Batch created successfully", newBatch)
}
