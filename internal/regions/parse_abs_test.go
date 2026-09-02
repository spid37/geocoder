package regions_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/spid37/geocoder/internal/regions"
	"github.com/xuri/excelize/v2"
	_ "modernc.org/sqlite"
)

func TestLoadMeshBlocks(t *testing.T) {
	dir := t.TempDir()
	xlsxPath := filepath.Join(dir, "MB_2021_AUST.xlsx")

	f := excelize.NewFile()
	sheet := "MB_2021_AUST"
	f.SetSheetName("Sheet1", sheet)
	headers := []string{"MB_CODE_2021", "SA3_CODE_2021", "SA3_NAME_2021"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	rows := [][]string{
		{"21402138100", "21402", "Mornington Peninsula"},
		{"20601110000", "20601", "Melbourne City"},
	}
	for r, row := range rows {
		for c, val := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			f.SetCellValue(sheet, cell, val)
		}
	}
	if err := f.SaveAs(xlsxPath); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		CREATE TABLE mesh_blocks (
			mb_code TEXT PRIMARY KEY,
			sa3_code TEXT,
			sa3_name TEXT
		)`); err != nil {
		t.Fatal(err)
	}

	if err := regions.LoadMeshBlocks(db, xlsxPath); err != nil {
		t.Fatal(err)
	}

	var sa3Code, sa3Name string
	err = db.QueryRow(`
		SELECT sa3_code, sa3_name
		FROM mesh_blocks WHERE mb_code = ?`, "21402138100").Scan(&sa3Code, &sa3Name)
	if err != nil {
		t.Fatal(err)
	}
	if sa3Code != "21402" || sa3Name != "Mornington Peninsula" {
		t.Fatalf("sa3: got %s %s", sa3Code, sa3Name)
	}
}
