package db

import "time"

type BalanceDAO struct {
	TokenName        string `gorm:"primaryKey;size:100;"`
	ContractAddress  string `gorm:"size:100;"`
	BlockHash        string `gorm:"size:100;"`
	BlockNumber      uint   `gorm:"primaryKey;"`
	LogIndex         uint   `gorm:"primaryKey;"`
	TransactionHash  string `gorm:"size:100;"`
	TransactionIndex uint   `gorm:"size:100;"`
	Address          string `gorm:"primaryKey;size:100;"`
	Balance          string `gorm:"size:100;"` // log index 后的最新余额

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

func (dao BalanceDAO) TableName() string {
	return "db_balances"
}
