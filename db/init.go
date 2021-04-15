package db

import (
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

func InitDB(mode string, source string) *gorm.DB {
	var (
		db  *gorm.DB
		err error
	)
	var gormLogger logger.Interface
	switch mode {
	case "debug":
		gormLogger = logger.Default.LogMode(logger.Info)
		logrus.SetReportCaller(true)
	default:
		gormLogger = nil

	}

	db, err = gorm.Open(mysql.Open(source), &gorm.Config{PrepareStmt: true, Logger: gormLogger})
	if err != nil {
		logrus.WithError(err).Fatal("failed to connect to database")
	}

	sqlDB, err := db.DB()

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	tables := []interface{}{
		//&DbContact{},
		&BalanceDAO{},
	}
	for _, table := range tables {
		err := db.AutoMigrate(table)
		if err != nil {
			logrus.WithError(err).Fatal("failed to migrate")
		}
	}
	logrus.Info("db init success")

	return db
}
