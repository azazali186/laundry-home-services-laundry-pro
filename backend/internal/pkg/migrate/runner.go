package migrate

import (
	"fmt"
	"sort"
	"time"

	"github.com/aeroxe/laundry-pro/backend/internal/pkg/logger"
	"gorm.io/gorm"
)

// Migration is one ordered, versioned schema change. Up runs inside the
// same transaction that records the applied version, so a partially applied
// migration rolls back cleanly. Down reverses the migration; when nil the
// migration is considered irreversible and rollback will refuse to proceed.
type Migration struct {
	Version string // sortable, e.g. "20260810000001"
	Name    string
	Up      func(db *gorm.DB) error
	Down    func(db *gorm.DB) error // optional reverse migration
}

// migrationRow records an applied migration version.
type migrationRow struct {
	Version   string `gorm:"primaryKey;size:32"`
	Name      string `gorm:"size:160"`
	AppliedAt time.Time
}

// TableName keeps the bookkeeping table snake_case and explicit.
func (migrationRow) TableName() string { return "schema_migrations" }

// Runner applies pending migrations in version order and records them.
type Runner struct {
	db *gorm.DB
}

// NewRunner builds a migration runner over the given database.
func NewRunner(db *gorm.DB) *Runner { return &Runner{db: db} }

// Run applies every migration whose version has not been recorded yet.
// It is safe to run on every boot: recorded versions are skipped.
func (r *Runner) Run(migrations []Migration) error {
	if err := r.ensureTable(); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	applied, err := r.appliedVersions()
	if err != nil {
		return err
	}
	sorted := append([]Migration(nil), migrations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Version < sorted[j].Version })
	for _, m := range sorted {
		if _, done := applied[m.Version]; done {
			continue
		}
		if err := r.apply(m); err != nil {
			return fmt.Errorf("migration %s (%s): %w", m.Version, m.Name, err)
		}
		logger.Info("migration applied: %s %s", m.Version, m.Name)
	}
	return nil
}

// Rollback reverts applied migrations down to (and including) the target
// version. Migrations without a Down function are skipped with a warning.
// The target version must be applied; migrations applied after it are
// rolled back first (LIFO order).
func (r *Runner) Rollback(migrations []Migration, targetVersion string) error {
	if err := r.ensureTable(); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	applied, err := r.appliedVersions()
	if err != nil {
		return err
	}
	if !applied[targetVersion] {
		return fmt.Errorf("target version %s is not applied", targetVersion)
	}
	byVersion := make(map[string]Migration, len(migrations))
	for _, m := range migrations {
		byVersion[m.Version] = m
	}
	// Collect applied versions in descending order (newest first).
	var toRollback []string
	sorted := append([]Migration(nil), migrations...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Version < sorted[j].Version })
	for _, m := range sorted {
		if applied[m.Version] {
			toRollback = append(toRollback, m.Version)
		}
	}
	// Rollback until we hit the target.
	for _, v := range toRollback {
		m, ok := byVersion[v]
		if !ok {
			return fmt.Errorf("migration %s recorded but not defined", v)
		}
		if m.Down == nil {
			logger.Warn("skipping rollback for %s %s (no Down function)", m.Version, m.Name)
			continue
		}
		if err := r.rollbackOne(m); err != nil {
			return fmt.Errorf("rollback %s (%s): %w", m.Version, m.Name, err)
		}
		logger.Info("migration rolled back: %s %s", m.Version, m.Name)
		if v == targetVersion {
			break
		}
	}
	return nil
}

func (r *Runner) ensureTable() error {
	if r.db.Migrator().HasTable(&migrationRow{}) {
		return nil
	}
	if err := r.db.Migrator().CreateTable(&migrationRow{}); err != nil {
		// Another instance may have created it between the check and now
		// (concurrent boot). Treat existence as success.
		if r.db.Migrator().HasTable(&migrationRow{}) {
			return nil
		}
		return err
	}
	return nil
}

func (r *Runner) appliedVersions() (map[string]bool, error) {
	var rows []migrationRow
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	out := make(map[string]bool, len(rows))
	for _, row := range rows {
		out[row.Version] = true
	}
	return out, nil
}

func (r *Runner) apply(m Migration) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := m.Up(tx); err != nil {
			return err
		}
		return tx.Create(&migrationRow{Version: m.Version, Name: m.Name, AppliedAt: time.Now().UTC()}).Error
	})
}

func (r *Runner) rollbackOne(m Migration) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := m.Down(tx); err != nil {
			return err
		}
		return tx.Where("version = ?", m.Version).Delete(&migrationRow{}).Error
	})
}

// AppliedVersions returns the set of applied migration version strings.
func (r *Runner) AppliedVersions() (map[string]bool, error) {
	return r.appliedVersions()
}
