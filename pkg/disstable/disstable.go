package disstable

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"cpdb2/pkg/db"
	"cpdb2/pkg/prompt"
)

const (
	DefaultSourceViewName = "sonority_dissonance_view"
	DefaultTargetTable    = "disstable"
)

// RecreateDissTable recreates the target table from the source view in the database.
func RecreateDissTable(ctx context.Context, database *db.DB, sourceView, targetTable string) error {
	if sourceView == "" {
		sourceView = DefaultSourceViewName
	}
	if targetTable == "" {
		targetTable = DefaultTargetTable
	}

	tx, err := database.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	columns, err := getViewColumns(ctx, tx, sourceView)
	if err != nil {
		return fmt.Errorf("get columns from %q: %w", sourceView, err)
	}
	selectQuery, err := buildTableSelect(columns, sourceView)
	if err != nil {
		return err
	}

	var existingTableName string
	err = tx.QueryRow(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = current_schema()
		  AND table_name = $1
		  AND table_type = 'BASE TABLE'
	`, targetTable).Scan(&existingTableName)

	if err == nil {
		backupName := fmt.Sprintf("%s_backup_%s", targetTable, time.Now().Format("20060102_150405"))

		_, err = tx.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s RENAME TO %s`, quoteIdent(targetTable), quoteIdent(backupName)))
		if err != nil {
			return fmt.Errorf("rename existing table: %w", err)
		}

		fmt.Printf("Renamed existing %q table to %q\n", targetTable, backupName)
	} else if !strings.Contains(err.Error(), "no rows in result set") {
		return fmt.Errorf("check existing table: %w", err)
	}

	_, err = tx.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE %s AS
		%s
		WITH NO DATA
	`, quoteIdent(targetTable), selectQuery))
	if err != nil {
		return fmt.Errorf("create fresh table: %w", err)
	}

	commandTag, err := tx.Exec(ctx, fmt.Sprintf(`
		INSERT INTO %s
		%s
	`, quoteIdent(targetTable), selectQuery))
	if err != nil {
		return fmt.Errorf("copy rows from view: %w", err)
	}

	fmt.Printf("Copied %d rows into %q\n", commandTag.RowsAffected(), targetTable)

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func getViewColumns(ctx context.Context, tx pgx.Tx, sourceView string) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = $1
		ORDER BY ordinal_position
	`, sourceView)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no columns found in view %q", sourceView)
	}
	return columns, nil
}

func buildTableSelect(columns []string, sourceView string) (string, error) {
	columnByName := make(map[string]string, len(columns))
	for _, column := range columns {
		columnByName[strings.ToLower(column)] = column
	}

	pieceColumn, ok := columnByName["piece"]
	if !ok {
		return "", fmt.Errorf("%q has no piece column", sourceView)
	}

	offsetColumn := ""
	for _, candidate := range []string{"offset", "startoffset", "toffset"} {
		if column, ok := columnByName[candidate]; ok {
			offsetColumn = column
			break
		}
	}
	if offsetColumn == "" {
		return "", fmt.Errorf("%q has no offset, startoffset, or toffset column", sourceView)
	}

	selectColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		if strings.EqualFold(column, "bar") {
			continue
		}
		selectColumns = append(selectColumns, "source."+quoteIdent(column))
	}
	if len(selectColumns) == 0 {
		return "", fmt.Errorf("%q has no columns other than bar", sourceView)
	}

	barColumn := fmt.Sprintf(`(
		SELECT n.bar
		FROM notes AS n
		WHERE n.piece = source.%s
		  AND n.toffset <= source.%s
		ORDER BY n.toffset DESC
		LIMIT 1
	) AS %s`, quoteIdent(pieceColumn), quoteIdent(offsetColumn), quoteIdent("bar"))
	selectColumns = append(selectColumns, barColumn)

	return fmt.Sprintf("SELECT %s FROM %s AS source", strings.Join(selectColumns, ", "), quoteIdent(sourceView)), nil
}

func quoteIdent(identifier string) string {
	quoted := `"`
	for _, r := range identifier {
		if r == '"' {
			quoted += `""`
		} else {
			quoted += string(r)
		}
	}
	quoted += `"`
	return quoted
}

// RunInteractive prompts the user and recreates the dissonance table.
func RunInteractive(ctx context.Context, database *db.DB, reader *bufio.Reader) error {
	sourceView, err := prompt.PromptString(reader, "Enter source view name", DefaultSourceViewName)
	if err != nil {
		return err
	}

	targetTable, err := prompt.PromptString(reader, "Enter target table name", DefaultTargetTable)
	if err != nil {
		return err
	}

	if err := RecreateDissTable(ctx, database, sourceView, targetTable); err != nil {
		return err
	}

	fmt.Printf("Successfully recreated %q from %q\n", targetTable, sourceView)
	return nil
}
