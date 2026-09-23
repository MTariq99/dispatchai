package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/examples/ecommerce/customers"
	ecomhttp "github.com/mtariq99/dispatchai/examples/ecommerce/http"
	"github.com/mtariq99/dispatchai/models"
)

func main() {
	cfg := &models.Config{}
	customersHandler := customers.NewCustomers(cfg)
	registry := map[string]ecomhttp.ToolHandler{
		"get_customers": customersHandler.GetCustomers,
	}

	handler := ecomhttp.NewHandler(cfg, registry)
	router := gin.Default()
	handler.RegisterRoutes(router)
	log.Println("example-project listening on :8081")
	if err := router.Run(":8081"); err != nil {
		log.Fatal(err)
	}
}
