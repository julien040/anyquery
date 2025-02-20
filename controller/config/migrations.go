package config

import (
	"database/sql"
	"fmt"
	"slices"
)

type migration struct {
	// The version of the migration
	Version int
	// The description of the migration
	Description string
	// The SQL queries to run for the migration
	Queries []string

	// A check to see if the migration has already been applied
	// This is used to skip migrations that have already been applied
	// If this is nil, the migration will always be applied
	Check func(db *sql.DB) (bool, error)
}

var migrations = []migration{
	{
		Version:     1,
		Description: "Add column tableMetadata to plugin_installed",
		Queries: []string{
			`ALTER TABLE plugin_installed ADD COLUMN tableMetadata TEXT DEFAULT '{}' NOT NULL`,
		},
		Check: func(db *sql.DB) (bool, error) {
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('plugin_installed') WHERE name = 'tableMetadata'").Scan(&count)
			if err != nil {
				return false, err
			}

			return count > 0, nil
		},
	},
}

// Apply migrations to the database
// unless they have already been applied
func applyMigrations(db *sql.DB) error {
	// Sort the migrations by semver
	slices.SortStableFunc(migrations, func(i, j migration) int {
		return i.Version - j.Version
	})

	// Get the current version
	var version int
	err := db.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("failed to get current database version: %w", err)
	}

	// Apply the migrations
	for _, m := range migrations {
		if version >= m.Version {
			continue
		}

		if m.Check != nil {
			alreadyApplied, err := m.Check(db)
			if err != nil {
				return fmt.Errorf("failed to check if migration %d has already been applied: %w", m.Version, err)
			}

			if alreadyApplied {
				continue
			}
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction for migration %d: %w", m.Version, err)
		}

		for _, q := range m.Queries {
			_, err := tx.Exec(q)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to apply migration %d: %w", m.Version, err)
			}
		}

		_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", m.Version))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update database version: %w", err)
		}

		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit transaction for migration %d: %w", m.Version, err)
		}

		version = m.Version
	}

	return nil
}
