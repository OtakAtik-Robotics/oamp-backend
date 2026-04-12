package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

func ExportExcel(c *gin.Context) {
	f := excelize.NewFile()
	defer f.Close()

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#CCCCCC"}},
	})

	sheetLeaderboard := "Leaderboard"
	f.SetSheetName("Sheet1", sheetLeaderboard)

	lbHeaders := []string{"Rank", "Name", "Grade", "Age", "VisuoSpatialFit", "TotalTime", "LevelReached", "DexterityScore"}
	for i, h := range lbHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetLeaderboard, cell, h)
	}
	f.SetCellStyle(sheetLeaderboard, "A1", "H1", headerStyle)

	entries := fetchLeaderboard(0)
	for i, e := range entries {
		row := i + 2
		f.SetCellValue(sheetLeaderboard, fmt.Sprintf("A%d", row), e.Rank)
		f.SetCellValue(sheetLeaderboard, fmt.Sprintf("B%d", row), e.Name)
		f.SetCellValue(sheetLeaderboard, fmt.Sprintf("C%d", row), e.Grade)
		f.SetCellValue(sheetLeaderboard, fmt.Sprintf("D%d", row), e.Age)
		f.SetCellValue(sheetLeaderboard, fmt.Sprintf("E%d", row), e.VisuoSpatialFit)
		f.SetCellValue(sheetLeaderboard, fmt.Sprintf("F%d", row), e.TotalTime)
		f.SetCellValue(sheetLeaderboard, fmt.Sprintf("G%d", row), e.LevelReached)
		f.SetCellValue(sheetLeaderboard, fmt.Sprintf("H%d", row), e.DexterityScore)
	}

	sheetParticipants := "Participants"
	f.NewSheet(sheetParticipants)

	pHeaders := []string{"ID", "UID", "Name", "Age", "Grade", "Gender", "Height", "Weight", "HeartRate", "SpO2", "GripStrength"}
	for i, h := range pHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetParticipants, cell, h)
	}
	f.SetCellStyle(sheetParticipants, "A1", "K1", headerStyle)

	var participants []model.Participant
	config.DB.Order("id asc").Find(&participants)

	for i, p := range participants {
		row := i + 2
		f.SetCellValue(sheetParticipants, fmt.Sprintf("A%d", row), p.ID)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("B%d", row), p.UID)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("C%d", row), p.Name)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("D%d", row), p.Age)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("E%d", row), p.Grade)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("F%d", row), p.Gender)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("G%d", row), p.Height)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("H%d", row), p.Weight)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("I%d", row), p.HeartRate)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("J%d", row), p.SpO2)
		f.SetCellValue(sheetParticipants, fmt.Sprintf("K%d", row), p.GripStrength)
	}

	sheetSessions := "Sessions"
	f.NewSheet(sheetSessions)

	sHeaders := []string{"ID", "ParticipantID", "Mode", "LevelReached", "TotalTime", "CognitiveAge", "VisuoSpatialFit", "DexterityScore"}
	for i, h := range sHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetSessions, cell, h)
	}
	f.SetCellStyle(sheetSessions, "A1", "H1", headerStyle)

	var sessions []model.GameSession
	config.DB.Order("id asc").Find(&sessions)

	for i, s := range sessions {
		row := i + 2
		f.SetCellValue(sheetSessions, fmt.Sprintf("A%d", row), s.ID)
		f.SetCellValue(sheetSessions, fmt.Sprintf("B%d", row), s.ParticipantID)
		f.SetCellValue(sheetSessions, fmt.Sprintf("C%d", row), s.Mode)
		f.SetCellValue(sheetSessions, fmt.Sprintf("D%d", row), s.LevelReached)
		f.SetCellValue(sheetSessions, fmt.Sprintf("E%d", row), s.TotalTime)
		f.SetCellValue(sheetSessions, fmt.Sprintf("F%d", row), s.CognitiveAge)
		f.SetCellValue(sheetSessions, fmt.Sprintf("G%d", row), s.VisuoSpatialFit)
		f.SetCellValue(sheetSessions, fmt.Sprintf("H%d", row), s.DexterityScore)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate Excel")
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=oamp-report.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

func ExportPDF(c *gin.Context) {
	entries := fetchLeaderboard(0)

	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(0, 10, "OAMP Leaderboard Report")
	pdf.Ln(14)

	if len(entries) == 0 {
		pdf.SetFont("Helvetica", "", 12)
		pdf.Cell(0, 10, "No game sessions recorded yet.")
	} else {
		headers := []string{"#", "Name", "Grade", "Age", "VisuoSpatial", "TotalTime", "Level", "Dexterity"}
		colWidths := []float64{10, 50, 20, 15, 35, 30, 20, 30}

		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetFillColor(200, 200, 200)
		for i, h := range headers {
			pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetFont("Helvetica", "", 10)
		for i, e := range entries {
			if i%2 == 0 {
				pdf.SetFillColor(240, 240, 240)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			row := []string{
				fmt.Sprintf("%d", e.Rank),
				e.Name,
				e.Grade,
				fmt.Sprintf("%d", e.Age),
				fmt.Sprintf("%.2f", e.VisuoSpatialFit),
				fmt.Sprintf("%.2f", e.TotalTime),
				fmt.Sprintf("%d", e.LevelReached),
				fmt.Sprintf("%.2f", e.DexterityScore),
			}
			for j, val := range row {
				pdf.CellFormat(colWidths[j], 7, val, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate PDF")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=oamp-leaderboard.pdf")
	c.Data(http.StatusOK, "application/pdf", buf.Bytes())
}
