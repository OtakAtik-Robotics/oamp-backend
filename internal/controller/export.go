package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"
	"regexp"
	"strings"

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

	entries := fetchLeaderboard(0, nil)
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
	entries := fetchLeaderboard(0, nil)

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

func ExportRapor(c *gin.Context) {
	uid := c.Param("uid")

	// Lookup participant
	var participant model.Participant
	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	// Get all sessions for this participant
	var sessions []model.GameSession
	config.DB.Where("participant_id = ?", participant.ID).Order("created_at asc").Find(&sessions)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetAutoPageBreak(true, 15)

	// --- Header ---
	pdf.SetFont("Helvetica", "B", 18)
	pdf.Cell(0, 10, "Rapor Peserta OAMP")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(120, 120, 120)
	pdf.Cell(0, 5, "OtakAtik-Robotics Event Report")
	pdf.Ln(10)

	// Divider line
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)

	// --- Participant Info ---
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.Cell(0, 8, participant.Name)
	pdf.Ln(8)

	pdf.SetFont("Helvetica", "", 10)
	info := [][]string{
		{"UID", participant.UID},
		{"Kelas", participant.Grade},
		{"Umur", fmt.Sprintf("%d tahun", participant.Age)},
		{"Jenis Kelamin", participant.Gender},
		{"Tinggi Badan", fmt.Sprintf("%.1f cm", participant.Height)},
		{"Berat Badan", fmt.Sprintf("%.1f kg", participant.Weight)},
		{"Detak Jantung", fmt.Sprintf("%d bpm", participant.HeartRate)},
		{"SpO2", fmt.Sprintf("%.1f%%", participant.SpO2)},
		{"Kekuatan Grip", fmt.Sprintf("%.1f kg", participant.GripStrength)},
	}

	labelW := 40.0
	valueW := 80.0
	for _, row := range info {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(labelW, 6, row[0], "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(valueW, 6, row[1], "", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(4)
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)

	// --- Game Sessions ---
	pdf.SetFont("Helvetica", "B", 13)
	pdf.Cell(0, 8, fmt.Sprintf("Riwayat Game (%d sesi)", len(sessions)))
	pdf.Ln(10)

	if len(sessions) == 0 {
		pdf.SetFont("Helvetica", "", 10)
		pdf.Cell(0, 6, "Belum ada sesi permainan.")
	} else {
		headers := []string{"#", "Tanggal", "Mode", "Level", "Waktu (s)", "VisuoSpatial", "Dexterity"}
		colW := []float64{8, 35, 18, 14, 20, 28, 28}

		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(66, 133, 244)
		pdf.SetTextColor(255, 255, 255)
		for i, h := range headers {
			pdf.CellFormat(colW[i], 7, h, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Helvetica", "", 9)

		// Compute summary
		var bestFit float64
		var totalTime float64
		var maxLevel int

		for i, s := range sessions {
			if s.VisuoSpatialFit > bestFit {
				bestFit = s.VisuoSpatialFit
			}
			totalTime += s.TotalTime
			if s.LevelReached > maxLevel {
				maxLevel = s.LevelReached
			}

			if i%2 == 0 {
				pdf.SetFillColor(245, 245, 245)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
			row := []string{
				fmt.Sprintf("%d", i+1),
				s.CreatedAt.Format("02/01/2006 15:04"),
				s.Mode,
				fmt.Sprintf("%d", s.LevelReached),
				fmt.Sprintf("%.1f", s.TotalTime),
				fmt.Sprintf("%.2f", s.VisuoSpatialFit),
				fmt.Sprintf("%.1f", s.DexterityScore),
			}
			for j, val := range row {
				pdf.CellFormat(colW[j], 6, val, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
		}

		// --- Summary ---
		avgTime := totalTime / float64(len(sessions))
		pdf.Ln(5)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.Cell(0, 7, "Ringkasan Performa")
		pdf.Ln(8)

		pdf.SetFont("Helvetica", "", 10)
		summary := [][]string{
			{"Total Sesi", fmt.Sprintf("%d", len(sessions))},
			{"Skor VisuoSpatial Terbaik", fmt.Sprintf("%.2f", bestFit)},
			{"Level Tertinggi", fmt.Sprintf("%d", maxLevel)},
			{"Rata-rata Waktu", fmt.Sprintf("%.1f detik", avgTime)},
		}
		for _, row := range summary {
			pdf.SetFont("Helvetica", "B", 10)
			pdf.CellFormat(60, 6, row[0], "", 0, "L", false, 0, "")
			pdf.SetFont("Helvetica", "", 10)
			pdf.CellFormat(40, 6, row[1], "", 0, "L", false, 0, "")
			pdf.Ln(-1)
		}
	}

	// --- Footer ---
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.Cell(0, 5, fmt.Sprintf("Dicetak pada %s — OAMP OtakAtik-Robotics", participant.CreatedAt.Format("02 January 2006")))

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate rapor")
		return
	}

	safeName := sanitizeFilename(participant.Name)
	filename := fmt.Sprintf("rapor-%s.pdf", safeName)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/pdf", buf.Bytes())
}

var nonAlphaNum = regexp.MustCompile(`[^\w\-]`)

func sanitizeFilename(name string) string {
	s := strings.TrimSpace(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlphaNum.ReplaceAllString(s, "")
	if s == "" {
		s = "unknown"
	}
	return s
}
