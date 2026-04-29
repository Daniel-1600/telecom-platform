package database

import (
	"context"

	"github.com/nutcas3/telecom-platform/apps/api-server/internal/models"
	"gorm.io/gorm"
)

// Subscriber CRUD
func (d *Database) CreateSubscriber(ctx context.Context, subscriber *models.Subscriber) error {
	return d.DB.WithContext(ctx).Create(subscriber).Error
}

func (d *Database) GetSubscriber(ctx context.Context, id uint) (*models.Subscriber, error) {
	var subscriber models.Subscriber
	err := d.DB.WithContext(ctx).Preload("Plan").First(&subscriber, id).Error
	if err != nil {
		return nil, err
	}
	return &subscriber, nil
}

func (d *Database) GetSubscriberByIMSI(ctx context.Context, imsi models.IMSI) (*models.Subscriber, error) {
	var subscriber models.Subscriber
	err := d.DB.WithContext(ctx).Preload("Plan").Where("imsi = ?", imsi).First(&subscriber).Error
	if err != nil {
		return nil, err
	}
	return &subscriber, nil
}

func (d *Database) UpdateSubscriber(ctx context.Context, subscriber *models.Subscriber) error {
	return d.DB.WithContext(ctx).Save(subscriber).Error
}

func (d *Database) ListSubscribers(ctx context.Context, req *ListSubscribersRequest) ([]models.Subscriber, int64, error) {
	var subscribers []models.Subscriber
	var total int64

	query := d.DB.WithContext(ctx).Model(&models.Subscriber{})
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.OrganizationID != "" {
		query = query.Where("organization_id = ?", req.OrganizationID)
	}
	if req.Search != "" {
		query = query.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ? OR msisdn ILIKE ?",
			"%"+req.Search+"%", "%"+req.Search+"%", "%"+req.Search+"%", "%"+req.Search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.PageSize
	err := query.Preload("Plan").Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&subscribers).Error
	return subscribers, total, err
}

func (d *Database) GetActiveSessionsByIMSI(ctx context.Context, imsi models.IMSI) ([]models.Session, error) {
	var sessions []models.Session
	err := d.DB.WithContext(ctx).Where("subscriber_id = ? AND status = ?", imsi, "active").Find(&sessions).Error
	return sessions, err
}

func (d *Database) UpdateSession(ctx context.Context, session *models.Session) error {
	return d.DB.WithContext(ctx).Save(session).Error
}

func (d *Database) UpdateSubscriberBalance(ctx context.Context, subscriberID uint, amount float64) error {
	return d.DB.WithContext(ctx).Model(&models.Subscriber{}).Where("id = ?", subscriberID).
		UpdateColumn("balance", gorm.Expr("balance + ?", amount)).Error
}

// Payment methods
func (d *Database) CreatePaymentMethod(ctx context.Context, pm *models.PaymentMethod) error {
	return d.DB.WithContext(ctx).Create(pm).Error
}

func (d *Database) GetPaymentMethod(ctx context.Context, id string) (*models.PaymentMethod, error) {
	var pm models.PaymentMethod
	err := d.DB.WithContext(ctx).Where("id = ?", id).First(&pm).Error
	if err != nil {
		return nil, err
	}
	return &pm, nil
}

func (d *Database) ListPaymentMethods(ctx context.Context, subscriberID uint) ([]models.PaymentMethod, error) {
	var methods []models.PaymentMethod
	err := d.DB.WithContext(ctx).Where("subscriber_id = ?", subscriberID).Find(&methods).Error
	return methods, err
}

func (d *Database) DeletePaymentMethod(ctx context.Context, id string) error {
	return d.DB.WithContext(ctx).Where("id = ?", id).Delete(&models.PaymentMethod{}).Error
}

// Transactions
func (d *Database) CreateTransaction(ctx context.Context, transaction *models.Transaction) error {
	return d.DB.WithContext(ctx).Create(transaction).Error
}

func (d *Database) GetTransaction(ctx context.Context, transactionID string) (*models.Transaction, error) {
	var tx models.Transaction
	err := d.DB.WithContext(ctx).Where("transaction_id = ?", transactionID).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (d *Database) GetTransactionByGatewayID(ctx context.Context, gatewayID string) (*models.Transaction, error) {
	var tx models.Transaction
	err := d.DB.WithContext(ctx).Where("transaction_id = ?", gatewayID).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (d *Database) GetTransactionByChargeID(ctx context.Context, chargeID string) (*models.Transaction, error) {
	var tx models.Transaction
	err := d.DB.WithContext(ctx).Where("transaction_id = ?", chargeID).First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (d *Database) UpdateTransaction(ctx context.Context, tx *models.Transaction) error {
	return d.DB.WithContext(ctx).Save(tx).Error
}

func (d *Database) ListTransactions(ctx context.Context, subscriberID uint) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := d.DB.WithContext(ctx).Where("subscriber_id = ?", subscriberID).Order("created_at DESC").Find(&transactions).Error
	return transactions, err
}

// Notifications and alerts
func (d *Database) CreateAlert(ctx context.Context, alert *models.Alert) error {
	return d.DB.WithContext(ctx).Create(alert).Error
}

func (d *Database) CreateNotification(ctx context.Context, notification *models.Notification) error {
	return d.DB.WithContext(ctx).Create(notification).Error
}

type ListSubscribersRequest struct {
	Page           int    `json:"page"`
	PageSize       int    `json:"page_size"`
	Status         string `json:"status"`
	OrganizationID string `json:"organization_id"`
	Search         string `json:"search"`
}
