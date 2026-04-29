package database

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/nutcas3/telecom-platform/apps/api-server/internal/config"
)

type Database struct {
	DB *gorm.DB
}

type IMSIAllocation struct {
	ID        uint   `gorm:"primaryKey"`
	LastIMSI  uint64 `gorm:"not null"`
	MinIMSI   uint64 `gorm:"not null"`
	MaxIMSI   uint64 `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host, cfg.Username, cfg.Password, cfg.Database, cfg.Port, cfg.SSLMode)

	gormLogger := logger.Default.LogMode(logger.Info)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	database := &Database{DB: db}

	if err := runMigrations(dsn); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := database.InitializeIMSIAllocation(); err != nil {
		return nil, fmt.Errorf("failed to initialize IMSI allocation: %w", err)
	}

	return database, nil
}

func runMigrations(dsn string) error {
	cmd := exec.Command("goose", "postgres", dsn, "up", "migrations")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("goose migration failed: %w, output: %s", err, string(output))
	}
	log.Printf("Database migrations completed successfully")
	return nil
}

func (d *Database) InitializeIMSIAllocation() error {
	var allocation IMSIAllocation
	result := d.DB.First(&allocation)

	if result.Error == gorm.ErrRecordNotFound {
		allocation = IMSIAllocation{LastIMSI: 0, MinIMSI: 1, MaxIMSI: 999999999}
		if err := d.DB.Create(&allocation).Error; err != nil {
			return fmt.Errorf("failed to create IMSI allocation: %w", err)
		}
		log.Printf("Created IMSI allocation record")
	} else if result.Error != nil {
		return fmt.Errorf("failed to query IMSI allocation: %w", result.Error)
	}

	return nil
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (d *Database) Ping(ctx context.Context) error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (d *Database) GetIMSIAllocation(ctx context.Context) (*IMSIAllocation, error) {
	var allocation IMSIAllocation
	err := d.DB.WithContext(ctx).First(&allocation).Error
	if err != nil {
		return nil, err
	}
	return &allocation, nil
}

func (d *Database) UpdateIMSIAllocation(ctx context.Context, allocation *IMSIAllocation) error {
	return d.DB.WithContext(ctx).Save(allocation).Error
}
