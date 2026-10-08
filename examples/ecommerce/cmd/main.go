package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/examples/ecommerce/customers"
	ecomhttp "github.com/mtariq99/dispatchai/examples/ecommerce/http"
	"github.com/mtariq99/dispatchai/examples/ecommerce/notifications"
	"github.com/mtariq99/dispatchai/examples/ecommerce/orders"
	"github.com/mtariq99/dispatchai/models"
)

func main() {
	cfg := &models.Config{}
	customersHandler := customers.NewCustomers(cfg)
	orderHandler := orders.NewOrders(cfg)
	notificationHandler := notifications.NewNotifications(cfg)
	registry := map[string]ecomhttp.ToolHandler{
		"get_customers":          customersHandler.GetCustomers,
		"find_delayed_customers": orderHandler.FindDelayedCustomers,
		"send_notification":      notificationHandler.SendNotification,
	}

	handler := ecomhttp.NewHandler(cfg, registry)
	router := gin.Default()
	handler.RegisterRoutes(router)
	log.Println("example-project listening on :8081")
	if err := router.Run(":8087"); err != nil {
		log.Fatal(err)
	}
}
