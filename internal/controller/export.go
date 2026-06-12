package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"oamp-backend/internal/config"
	"oamp-backend/internal/model"
	"oamp-backend/pkg/response"
	"os"
	"regexp"
	"strings"
	"time"

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

	entries := fetchLeaderboard(0, nil, "")
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

	pHeaders := []string{"ID", "UID", "Name", "Age", "Grade", "Gender", "Height", "Weight", "HeartRate", "GripStrength"}
	for i, h := range pHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetParticipants, cell, h)
	}
	f.SetCellStyle(sheetParticipants, "A1", "J1", headerStyle)

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
		f.SetCellValue(sheetParticipants, fmt.Sprintf("J%d", row), p.GripStrength)
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

	// ── Game Results (raw per-level data for analysis) ────────────────────
	sheetResults := "GameResults"
	f.NewSheet(sheetResults)

	rHeaders := []string{"UID", "Mode", "NickName", "Gender", "Age", "Task01", "Task02", "Task03", "Task04", "Task05", "Task06", "Task07", "Task08", "TaskAvg", "CognitiveAge", "VisuoSpatial", "CogAgeList", "VariantList", "ClientTs", "CreatedAt"}
	for i, h := range rHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetResults, cell, h)
	}
	f.SetCellStyle(sheetResults, "A1", fmt.Sprintf("%c1", 'A'+len(rHeaders)-1), headerStyle)

	var results []model.GameResult
	config.DB.Order("created_at desc").Find(&results)

	for i, r := range results {
		row := i + 2
		varJSON, _ := json.Marshal(r.VariantList)
		cogJSON, _ := json.Marshal(r.CogAgeList)
		f.SetCellValue(sheetResults, fmt.Sprintf("A%d", row), r.UID)
		f.SetCellValue(sheetResults, fmt.Sprintf("B%d", row), r.Mode)
		f.SetCellValue(sheetResults, fmt.Sprintf("C%d", row), r.NickName)
		f.SetCellValue(sheetResults, fmt.Sprintf("D%d", row), r.Gender)
		f.SetCellValue(sheetResults, fmt.Sprintf("E%d", row), r.Age)
		f.SetCellValue(sheetResults, fmt.Sprintf("F%d", row), r.Task01)
		f.SetCellValue(sheetResults, fmt.Sprintf("G%d", row), r.Task02)
		f.SetCellValue(sheetResults, fmt.Sprintf("H%d", row), r.Task03)
		f.SetCellValue(sheetResults, fmt.Sprintf("I%d", row), r.Task04)
		f.SetCellValue(sheetResults, fmt.Sprintf("J%d", row), r.Task05)
		f.SetCellValue(sheetResults, fmt.Sprintf("K%d", row), r.Task06)
		f.SetCellValue(sheetResults, fmt.Sprintf("L%d", row), r.Task07)
		f.SetCellValue(sheetResults, fmt.Sprintf("M%d", row), r.Task08)
		f.SetCellValue(sheetResults, fmt.Sprintf("N%d", row), r.TaskAvg)
		f.SetCellValue(sheetResults, fmt.Sprintf("O%d", row), r.CognitiveAge)
		f.SetCellValue(sheetResults, fmt.Sprintf("P%d", row), r.VisuoSpatial)
		f.SetCellValue(sheetResults, fmt.Sprintf("Q%d", row), string(cogJSON))
		f.SetCellValue(sheetResults, fmt.Sprintf("R%d", row), string(varJSON))
		f.SetCellValue(sheetResults, fmt.Sprintf("S%d", row), r.ClientTs)
		f.SetCellValue(sheetResults, fmt.Sprintf("T%d", row), r.CreatedAt.Format(time.RFC3339))
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
	entries := fetchLeaderboard(0, nil, "")

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

	var participant model.Participant
	if err := config.DB.Where("uid = ?", uid).First(&participant).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Participant not found")
		return
	}

	var sessions []model.GameSession
	config.DB.Where("participant_id = ?", participant.ID).Order("created_at asc").Find(&sessions)

	var result model.GameResult
	config.DB.Where("uid = ?", uid).First(&result)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetAutoPageBreak(true, 15)

	// ── Branded Header ──────────────────────────────────────────────────
	red := [3]int{220, 38, 38}
	pdf.SetFillColor(red[0], red[1], red[2])
	pdf.Rect(0, 0, 210, 35, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetY(8)
	pdf.CellFormat(0, 10, "🧩  Otak Atik Merah Putih", "", 0, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetY(20)
	pdf.CellFormat(0, 5, "Block Design Test — Kartu Hasil Peserta", "", 0, "C", false, 0, "")
	pdf.Ln(40)

	// ── Avatar Circle ───────────────────────────────────────────────────
	initial := strings.ToUpper(string(participant.Name[0]))
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(red[0], red[1], red[2])
	pdf.Circle(105, 52, 10, "F")
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetY(48)
	pdf.CellFormat(0, 10, initial, "", 0, "C", false, 0, "")
	pdf.Ln(14)

	// ── Name + Meta ─────────────────────────────────────────────────────
	pdf.SetTextColor(30, 30, 30)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 8, participant.Name, "", 0, "C", false, 0, "")
	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(0, 5, fmt.Sprintf("%s · %d tahun · %s · UID: %s", participant.Grade, participant.Age, participant.Gender, participant.UID), "", 0, "C", false, 0, "")
	pdf.Ln(10)

	// ── KPI Card ────────────────────────────────────────────────────────
	var bestScore float64
	var maxLevel int
	var avgCognitive int
	for _, s := range sessions {
		if s.Score > bestScore { bestScore = s.Score }
		if s.LevelReached > maxLevel { maxLevel = s.LevelReached }
	}
	if result.UID != "" {
		avgCognitive = int(result.CognitiveAge)
	}
	pdf.SetDrawColor(red[0], red[1], red[2])
	pdf.RoundedRect(15, pdf.GetY(), 180, 25, 3, "1234", "D")
	pdf.SetFillColor(254, 242, 242) // light red
	pdf.Rect(15, pdf.GetY(), 180, 25, "F")

	kpiLabels := []string{"Skor Terbaik", "Level", "Usia Kognitif", "Visuo-Spatial"}
	kpiValues := []string{
		fmt.Sprintf("%.0f pts", bestScore),
		fmt.Sprintf("%d/8", maxLevel),
		fmt.Sprintf("%d th", avgCognitive),
		fmt.Sprintf("%.0f%%", result.VisuoSpatial),
	}
	kpiX := 15.0
	for i := 0; i < 4; i++ {
		pdf.SetFillColor(red[0], red[1], red[2])
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetXY(kpiX, pdf.GetY()+2)
		pdf.CellFormat(42, 5, kpiLabels[i], "", 0, "C", false, 0, "")
		pdf.SetTextColor(30, 30, 30)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetXY(kpiX, pdf.GetY()+5.5)
		pdf.CellFormat(42, 6, kpiValues[i], "", 0, "C", false, 0, "")
		kpiX += 45
	}
	pdf.Ln(30)

	// ── Biodata ─────────────────────────────────────────────────────────
	pdf.SetTextColor(30, 30, 30)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.Cell(0, 7, "Biodata")
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 10)
	bio := [][]string{
		{"Tinggi Badan", fmt.Sprintf("%.1f cm", participant.Height)},
		{"Berat Badan", fmt.Sprintf("%.1f kg", participant.Weight)},
		{"Kekuatan Grip", fmt.Sprintf("%.1f kg", participant.GripStrength)},
		{"Dexterity", fmt.Sprintf("%.0f", participant.Dexterity)},
	}
	for _, row := range bio {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(40, 5, row[0], "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(50, 5, row[1], "", 0, "L", false, 0, "")
		pdf.Ln(6)
	}
	pdf.Ln(2)

	// ── Per-Level Breakdown ─────────────────────────────────────────────
	if result.UID != "" && (result.Task01+result.Task02+result.Task03+result.Task04+result.Task05+result.Task06+result.Task07+result.Task08) > 0 {
		pdf.SetFont("Helvetica", "B", 11)
		pdf.Cell(0, 7, "Waktu Per Level")
		pdf.Ln(8)
		tasks := []float64{result.Task01, result.Task02, result.Task03, result.Task04, result.Task05, result.Task06, result.Task07, result.Task08}
		for i := 0; i < 8; i++ {
			col := i % 4
			if col == 0 { pdf.SetX(15) }
			pdf.SetFont("Helvetica", "B", 9)
			pdf.SetTextColor(red[0], red[1], red[2])
			pdf.CellFormat(10, 6, fmt.Sprintf("L%d", i+1), "", 0, "L", false, 0, "")
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(30, 30, 30)
			pdf.CellFormat(32, 6, fmt.Sprintf("%.2fs", tasks[i]), "", 0, "L", false, 0, "")
			if col == 3 { pdf.Ln(7) }
		}
		pdf.Ln(12)
	}

	// ── Session History ─────────────────────────────────────────────────
	if len(sessions) > 0 {
		pdf.SetFont("Helvetica", "B", 11)
		pdf.Cell(0, 7, fmt.Sprintf("Riwayat Permainan (%d sesi)", len(sessions)))
		pdf.Ln(9)

		hdr := []string{"#", "Tanggal", "Mode", "Level", "Skor"}
		hdrW := []float64{8, 40, 22, 15, 25}
		pdf.SetFillColor(red[0], red[1], red[2])
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 9)
		for i, h := range hdr {
			pdf.CellFormat(hdrW[i], 7, h, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		pdf.SetTextColor(30, 30, 30)
		pdf.SetFont("Helvetica", "", 9)
		for i, s := range sessions {
			if i%2 == 0 { pdf.SetFillColor(254, 242, 242) } else { pdf.SetFillColor(255, 255, 255) }
			row := []string{
				fmt.Sprintf("%d", i+1),
				s.CreatedAt.Format("02/01/06 15:04"),
				s.Mode,
				fmt.Sprintf("%d", s.LevelReached),
				fmt.Sprintf("%.0f", s.Score),
			}
			for j, val := range row {
				pdf.CellFormat(hdrW[j], 6, val, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
		}
	} else {
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(150, 150, 150)
		pdf.Cell(0, 6, "Belum ada sesi permainan.")
	}

	// ── Footer ──────────────────────────────────────────────────────────
	pdf.Ln(8)
	pdf.SetDrawColor(red[0], red[1], red[2])
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(4)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 5, fmt.Sprintf("Otak Atik Merah Putih © %d — Dicetak pada %s", time.Now().Year(), time.Now().Format("02 Jan 2006 15:04")), "", 0, "C", false, 0, "")

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

// ExportCSV — GET /api/v1/export/csv — downloadable ZIP of all data tables as CSV
func ExportCSV(c *gin.Context) {
	f := excelize.NewFile()
	defer f.Close()

	// Sheet: GameResults (raw per-level data)
	f.SetSheetName("Sheet1", "GameResults")
	rHeaders := []string{"UID","Mode","NickName","Gender","Age","Task01","Task02","Task03","Task04","Task05","Task06","Task07","Task08","TaskAvg","CognitiveAge","VisuoSpatial","CogAgeList","VariantList","ClientTs","CreatedAt"}
	for i, h := range rHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("GameResults", cell, h)
	}
	var results []model.GameResult
	config.DB.Order("created_at desc").Find(&results)
	for i, r := range results {
		row := i + 2
		varJSON, _ := json.Marshal(r.VariantList)
		cogJSON, _ := json.Marshal(r.CogAgeList)
		f.SetCellValue("GameResults", fmt.Sprintf("A%d", row), r.UID)
		f.SetCellValue("GameResults", fmt.Sprintf("B%d", row), r.Mode)
		f.SetCellValue("GameResults", fmt.Sprintf("C%d", row), r.NickName)
		f.SetCellValue("GameResults", fmt.Sprintf("D%d", row), r.Gender)
		f.SetCellValue("GameResults", fmt.Sprintf("E%d", row), r.Age)
		f.SetCellValue("GameResults", fmt.Sprintf("F%d", row), r.Task01)
		f.SetCellValue("GameResults", fmt.Sprintf("G%d", row), r.Task02)
		f.SetCellValue("GameResults", fmt.Sprintf("H%d", row), r.Task03)
		f.SetCellValue("GameResults", fmt.Sprintf("I%d", row), r.Task04)
		f.SetCellValue("GameResults", fmt.Sprintf("J%d", row), r.Task05)
		f.SetCellValue("GameResults", fmt.Sprintf("K%d", row), r.Task06)
		f.SetCellValue("GameResults", fmt.Sprintf("L%d", row), r.Task07)
		f.SetCellValue("GameResults", fmt.Sprintf("M%d", row), r.Task08)
		f.SetCellValue("GameResults", fmt.Sprintf("N%d", row), r.TaskAvg)
		f.SetCellValue("GameResults", fmt.Sprintf("O%d", row), r.CognitiveAge)
		f.SetCellValue("GameResults", fmt.Sprintf("P%d", row), r.VisuoSpatial)
		f.SetCellValue("GameResults", fmt.Sprintf("Q%d", row), string(cogJSON))
		f.SetCellValue("GameResults", fmt.Sprintf("R%d", row), string(varJSON))
		f.SetCellValue("GameResults", fmt.Sprintf("S%d", row), r.ClientTs)
		f.SetCellValue("GameResults", fmt.Sprintf("T%d", row), r.CreatedAt.Format(time.RFC3339))
	}

	// Sheet: Sessions
	f.NewSheet("Sessions")
	sHeaders := []string{"ID","ParticipantID","Mode","LevelReached","TotalTime","CognitiveAge","VisuoSpatialFit","DexterityScore","Score","CreatedAt"}
	for i, h := range sHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Sessions", cell, h)
	}
	var sessions []model.GameSession
	config.DB.Order("id asc").Find(&sessions)
	for i, s := range sessions {
		row := i + 2
		f.SetCellValue("Sessions", fmt.Sprintf("A%d", row), s.ID)
		f.SetCellValue("Sessions", fmt.Sprintf("B%d", row), s.ParticipantID)
		f.SetCellValue("Sessions", fmt.Sprintf("C%d", row), s.Mode)
		f.SetCellValue("Sessions", fmt.Sprintf("D%d", row), s.LevelReached)
		f.SetCellValue("Sessions", fmt.Sprintf("E%d", row), s.TotalTime)
		f.SetCellValue("Sessions", fmt.Sprintf("F%d", row), s.CognitiveAge)
		f.SetCellValue("Sessions", fmt.Sprintf("G%d", row), s.VisuoSpatialFit)
		f.SetCellValue("Sessions", fmt.Sprintf("H%d", row), s.DexterityScore)
		f.SetCellValue("Sessions", fmt.Sprintf("I%d", row), s.Score)
		f.SetCellValue("Sessions", fmt.Sprintf("J%d", row), s.CreatedAt.Format(time.RFC3339))
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate export")
		return
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=oamp-full-export.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// SendExportToTelegram — POST /api/v1/export/telegram — generate Excel + send via Telegram
func SendExportToTelegram(c *gin.Context) {
	token := getEnv("TELEGRAM_BOT_TOKEN", "")
	chatID := getEnv("TELEGRAM_CHAT_ID", "")
	if token == "" || chatID == "" {
		response.Error(c, http.StatusBadRequest, "Telegram not configured")
		return
	}

	// Generate Excel in memory
	f := excelize.NewFile()
	defer f.Close()

	f.SetSheetName("Sheet1", "GameResults")
	rHeaders := []string{"UID","Mode","NickName","Gender","Age","Task01","Task02","Task03","Task04","Task05","Task06","Task07","Task08","TaskAvg","CognitiveAge","VisuoSpatial"}
	for i, h := range rHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("GameResults", cell, h)
	}
	var results []model.GameResult
	config.DB.Order("created_at desc").Find(&results)
	for i, r := range results {
		row := i + 2
		f.SetCellValue("GameResults", fmt.Sprintf("A%d", row), r.UID)
		f.SetCellValue("GameResults", fmt.Sprintf("B%d", row), r.Mode)
		f.SetCellValue("GameResults", fmt.Sprintf("C%d", row), r.NickName)
		f.SetCellValue("GameResults", fmt.Sprintf("D%d", row), r.Gender)
		f.SetCellValue("GameResults", fmt.Sprintf("E%d", row), r.Age)
		f.SetCellValue("GameResults", fmt.Sprintf("F%d", row), r.Task01)
		f.SetCellValue("GameResults", fmt.Sprintf("G%d", row), r.Task02)
		f.SetCellValue("GameResults", fmt.Sprintf("H%d", row), r.Task03)
		f.SetCellValue("GameResults", fmt.Sprintf("I%d", row), r.Task04)
		f.SetCellValue("GameResults", fmt.Sprintf("J%d", row), r.Task05)
		f.SetCellValue("GameResults", fmt.Sprintf("K%d", row), r.Task06)
		f.SetCellValue("GameResults", fmt.Sprintf("L%d", row), r.Task07)
		f.SetCellValue("GameResults", fmt.Sprintf("M%d", row), r.Task08)
		f.SetCellValue("GameResults", fmt.Sprintf("N%d", row), r.TaskAvg)
		f.SetCellValue("GameResults", fmt.Sprintf("O%d", row), r.CognitiveAge)
		f.SetCellValue("GameResults", fmt.Sprintf("P%d", row), r.VisuoSpatial)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate Excel")
		return
	}

	// Send via Telegram (async, fire-and-forget)
	go func(data []byte) {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", token)
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		w.WriteField("chat_id", chatID)
		w.WriteField("caption", "📊 OAMP Full Export — Game Results")
		part, _ := w.CreateFormFile("document", "oamp-export.xlsx")
		part.Write(data)
		w.Close()

		resp, err := http.Post(url, w.FormDataContentType(), &b)
		if err != nil {
			log.Printf("[telegram] export send failed: %v", err)
			return
		}
		resp.Body.Close()
		log.Printf("[telegram] export sent to chat %s", chatID)
	}(buf.Bytes())

	response.OKWithMessage(c, "Mengirim export ke Telegram...", nil)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
