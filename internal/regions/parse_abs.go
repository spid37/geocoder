package regions

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

func LoadMeshBlocks(db *sql.DB, xlsxPath string) error {
	f, err := excelize.OpenFile(xlsxPath)
	if err != nil {
		return fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	for _, name := range f.GetSheetList() {
		if strings.EqualFold(name, "MB_2021_AUST") {
			sheet = name
			break
		}
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return err
	}
	if len(rows) < 2 {
		return fmt.Errorf("empty mesh block sheet")
	}

	header := map[string]int{}
	for i, col := range rows[0] {
		header[strings.ToUpper(strings.TrimSpace(col))] = i
	}

	mbCol := colIndex(header, "MB_CODE_2021")
	sa3CodeCol := colIndex(header, "SA3_CODE_2021")
	sa3NameCol := colIndex(header, "SA3_NAME_2021")
	if mbCol < 0 || sa3CodeCol < 0 || sa3NameCol < 0 {
		return fmt.Errorf("missing required columns in %s", sheet)
	}

	if _, err := db.Exec(`DELETE FROM mesh_blocks`); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO mesh_blocks (mb_code, sa3_code, sa3_name)
		VALUES (?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}

	count := 0
	for _, row := range rows[1:] {
		mb := cell(row, mbCol)
		if mb == "" {
			continue
		}
		if _, err := stmt.Exec(
			mb,
			cell(row, sa3CodeCol),
			cell(row, sa3NameCol),
		); err != nil {
			stmt.Close()
			tx.Rollback()
			return err
		}
		count++
		if count%5000 == 0 {
			stmt.Close()
			if err := tx.Commit(); err != nil {
				return err
			}
			tx, err = db.Begin()
			if err != nil {
				return err
			}
			stmt, err = tx.Prepare(`
				INSERT OR REPLACE INTO mesh_blocks (mb_code, sa3_code, sa3_name)
				VALUES (?, ?, ?)`)
			if err != nil {
				return err
			}
		}
	}

	stmt.Close()
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("Loaded %d mesh blocks\n", count)
	return nil
}

func colIndex(header map[string]int, name string) int {
	if idx, ok := header[name]; ok {
		return idx
	}
	return -1
}

func cell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
