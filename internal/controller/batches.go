package controller

import (
	"fmt"
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type CreateBatchPayload struct {
	Name string `json:"name" binding:"required"`
}

type RenameBatchPayload struct {
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

	if err := tx.Model(&model.EventBatch{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "Failed to deactivate existing batches")
		return
	}

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

func RenameBatch(c *gin.Context) {
	id := c.Param("id")

	var payload RenameBatchPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.Error(c, http.StatusBadRequest, response.FormatBindError(err))
		return
	}

	result := config.DB.Model(&model.EventBatch{}).Where("id = ?", id).Update("name", payload.Name)
	if result.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to rename batch")
		return
	}
	if result.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Batch not found")
		return
	}

	response.OKWithMessage(c, fmt.Sprintf("Batch renamed to %s", payload.Name), nil)
}

func DeleteBatch(c *gin.Context) {
	id := c.Param("id")

	var batch model.EventBatch
	if err := config.DB.First(&batch, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Batch not found")
		return
	}

	var sessionCount int64
	config.DB.Model(&model.GameSession{}).Where("event_batch_id = ?", id).Count(&sessionCount)

	if err := config.DB.Delete(&batch).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to delete batch")
		return
	}

	response.OKWithMessage(c, fmt.Sprintf("Batch deleted (%d sessions cleared)", sessionCount), nil)
}

func ActivateBatch(c *gin.Context) {
	id := c.Param("id")

	var batch model.EventBatch
	if err := config.DB.First(&batch, id).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Batch not found")
		return
	}

	tx := config.DB.Begin()

	if err := tx.Model(&model.EventBatch{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "Failed to deactivate other batches")
		return
	}

	if err := tx.Model(&batch).Update("is_active", true).Error; err != nil {
		tx.Rollback()
		response.Error(c, http.StatusInternalServerError, "Failed to activate batch")
		return
	}

	tx.Commit()
	response.OKWithMessage(c, fmt.Sprintf("Batch %s is now active", batch.Name), nil)
}
