package models

import "time"

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusConfirmed OrderStatus = "confirmed"
	StatusShipped   OrderStatus = "shipped"
	StatusDelivered OrderStatus = "delivered"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	CustomerID  uint        `json:"customer_id" gorm:"not null"`
	Customer    Customer    `json:"customer" gorm:"foreignKey:CustomerID"`
	Status      OrderStatus `json:"status" gorm:"default:pending"`
	TotalAmount float64     `json:"total_amount"`
	SlipImage   string      `json:"slip_image"`
	Notes       string      `json:"notes"`
	Items       []OrderItem `json:"items" gorm:"foreignKey:OrderID"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID        uint    `json:"id" gorm:"primaryKey"`
	OrderID   uint    `json:"order_id" gorm:"not null"`
	ProductID uint    `json:"product_id" gorm:"not null"`
	Product   Product `json:"product" gorm:"foreignKey:ProductID"`
	Quantity  int     `json:"quantity" gorm:"not null"`
	Price     float64 `json:"price" gorm:"not null"`
}

type CreateOrderRequest struct {
	CustomerID uint               `json:"customer_id"`
	Notes      string             `json:"notes"`
	Items      []CreateOrderItem  `json:"items"`
}

type CreateOrderItem struct {
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}
