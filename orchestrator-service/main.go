package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type OrchestrationRequest struct {
	CustomerID  int         `json:"customer_id"`
	Items       []OrderItem `json:"items"`
	VoucherCode string      `json:"voucher_code,omitempty"`
}

type OrderItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

type OrchestrationResponse struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	OrderID     int     `json:"order_id,omitempty"`
	SalesID     int     `json:"sales_id,omitempty"`
	FinalAmount float64 `json:"final_amount,omitempty"`
	Error       string  `json:"error,omitempty"`
}

type WorkflowStep struct {
	Name     string      `json:"name"`
	Status   string      `json:"status"`
	Response interface{} `json:"response,omitempty"`
	Error    string      `json:"error,omitempty"`
}

type Workflow struct {
	ID       string                `json:"id"`
	Status   string                `json:"status"`
	Steps    []WorkflowStep        `json:"steps"`
	Request  OrchestrationRequest  `json:"request"`
	Response OrchestrationResponse `json:"response"`
}

var workflows = make(map[string]*Workflow)

func main() {
	godotenv.Load()
	
	r := gin.Default()
	
	r.POST("/orchestrate/order", orchestrateOrder)
	r.GET("/orchestrate/:id", getOrchestrationStatus)
	r.POST("/orchestrate/:id/compensate", compensateWorkflow)
	
	log.Println("Orchestrator service running on port 8081")
	r.Run(":8081")
}

func orchestrateOrder(c *gin.Context) {
	var req OrchestrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	workflowID := fmt.Sprintf("wf_%d", time.Now().UnixNano())
	workflow := &Workflow{
		ID:      workflowID,
		Status:  "running",
		Request: req,
		Steps:   []WorkflowStep{},
	}
	
	workflows[workflowID] = workflow
	
	go executeWorkflow(workflow)
	
	c.JSON(202, gin.H{
		"workflow_id": workflowID,
		"status":      "accepted",
	})
}

func executeWorkflow(workflow *Workflow) {
	defer func() {
		if r := recover(); r != nil {
			workflow.Status = "failed"
			workflow.Response.Error = fmt.Sprintf("Workflow panic: %v", r)
		}
	}()
	
	// Step 1: Create Order
	orderStep := WorkflowStep{Name: "create_order", Status: "running"}
	workflow.Steps = append(workflow.Steps, orderStep)
	
	orderResp, err := createOrder(workflow.Request)
	if err != nil {
		workflow.Steps[len(workflow.Steps)-1].Status = "failed"
		workflow.Steps[len(workflow.Steps)-1].Error = err.Error()
		workflow.Status = "failed"
		workflow.Response.Error = err.Error()
		return
	}
	
	workflow.Steps[len(workflow.Steps)-1].Status = "completed"
	workflow.Steps[len(workflow.Steps)-1].Response = orderResp
	workflow.Response.OrderID = orderResp.ID
	
	// Step 2: Process Sales Transaction
	salesStep := WorkflowStep{Name: "process_sales", Status: "running"}
	workflow.Steps = append(workflow.Steps, salesStep)
	
	salesResp, err := processSales(workflow.Request, orderResp.ID, orderResp.TotalAmount)
	if err != nil {
		workflow.Steps[len(workflow.Steps)-1].Status = "failed"
		workflow.Steps[len(workflow.Steps)-1].Error = err.Error()
		
		compensateOrder(orderResp.ID)
		
		workflow.Status = "compensated"
		workflow.Response.Error = err.Error()
		return
	}
	
	workflow.Steps[len(workflow.Steps)-1].Status = "completed"
	workflow.Steps[len(workflow.Steps)-1].Response = salesResp
	workflow.Response.SalesID = salesResp.ID
	workflow.Response.FinalAmount = salesResp.FinalAmount
	
	// Step 3: Confirm Order
	confirmStep := WorkflowStep{Name: "confirm_order", Status: "running"}
	workflow.Steps = append(workflow.Steps, confirmStep)
	
	err = confirmOrder(orderResp.ID)
	if err != nil {
		workflow.Steps[len(workflow.Steps)-1].Status = "failed"
		workflow.Steps[len(workflow.Steps)-1].Error = err.Error()
		
		compensateSales(salesResp.ID)
		compensateOrder(orderResp.ID)
		
		workflow.Status = "compensated"
		workflow.Response.Error = err.Error()
		return
	}
	
	workflow.Steps[len(workflow.Steps)-1].Status = "completed"
	workflow.Status = "completed"
	workflow.Response.Status = "success"
}

type OrderResponse struct {
	ID          int     `json:"id"`
	TotalAmount float64 `json:"total_amount"`
	Status      string  `json:"status"`
}

type SalesResponse struct {
	ID          int     `json:"id"`
	FinalAmount float64 `json:"final_amount"`
	Status      string  `json:"status"`
}

func createOrder(req OrchestrationRequest) (*OrderResponse, error) {
	orderServiceURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8080")
	
	orderReq := map[string]interface{}{
		"customer_id": req.CustomerID,
		"items":       req.Items,
	}
	
	jsonData, _ := json.Marshal(orderReq)
	resp, err := http.Post(orderServiceURL+"/orders", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 201 {
		return nil, fmt.Errorf("order service returned status %d", resp.StatusCode)
	}
	
	var orderResp OrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %v", err)
	}
	
	return &orderResp, nil
}

func processSales(req OrchestrationRequest, orderID int, totalAmount float64) (*SalesResponse, error) {
	salesServiceURL := getEnv("SALES_SERVICE_URL", "http://localhost:3000")
	
	salesReq := map[string]interface{}{
		"order_id":        orderID,
		"customer_id":     req.CustomerID,
		"original_amount": totalAmount,
		"voucher_code":    req.VoucherCode,
	}
	
	jsonData, _ := json.Marshal(salesReq)
	resp, err := http.Post(salesServiceURL+"/sales/process", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to process sales: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("sales service returned status %d", resp.StatusCode)
	}
	
	var salesResp SalesResponse
	if err := json.NewDecoder(resp.Body).Decode(&salesResp); err != nil {
		return nil, fmt.Errorf("failed to decode sales response: %v", err)
	}
	
	return &salesResp, nil
}

func confirmOrder(orderID int) error {
	orderServiceURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8080")
	
	statusReq := map[string]string{"status": "completed"}
	jsonData, _ := json.Marshal(statusReq)
	
	client := &http.Client{}
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/orders/%d/status", orderServiceURL, orderID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to confirm order: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("order service returned status %d", resp.StatusCode)
	}
	
	return nil
}

func compensateOrder(orderID int) error {
	orderServiceURL := getEnv("ORDER_SERVICE_URL", "http://localhost:8080")
	
	statusReq := map[string]string{"status": "cancelled"}
	jsonData, _ := json.Marshal(statusReq)
	
	client := &http.Client{}
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/orders/%d/status", orderServiceURL, orderID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to compensate order %d: %v", orderID, err)
		return err
	}
	defer resp.Body.Close()
	
	return nil
}

func compensateSales(salesID int) error {
	log.Printf("Compensating sales transaction %d", salesID)
	return nil
}

func getOrchestrationStatus(c *gin.Context) {
	workflowID := c.Param("id")
	
	workflow, exists := workflows[workflowID]
	if !exists {
		c.JSON(404, gin.H{"error": "Workflow not found"})
		return
	}
	
	c.JSON(200, workflow)
}

func compensateWorkflow(c *gin.Context) {
	workflowID := c.Param("id")
	
	workflow, exists := workflows[workflowID]
	if !exists {
		c.JSON(404, gin.H{"error": "Workflow not found"})
		return
	}
	
	if workflow.Status != "failed" {
		c.JSON(400, gin.H{"error": "Can only compensate failed workflows"})
		return
	}
	
	for i := len(workflow.Steps) - 1; i >= 0; i-- {
		step := workflow.Steps[i]
		if step.Status == "completed" {
			switch step.Name {
			case "confirm_order":
				if workflow.Response.OrderID > 0 {
					compensateOrder(workflow.Response.OrderID)
				}
			case "process_sales":
				if workflow.Response.SalesID > 0 {
					compensateSales(workflow.Response.SalesID)
				}
			case "create_order":
				if workflow.Response.OrderID > 0 {
					compensateOrder(workflow.Response.OrderID)
				}
			}
		}
	}
	
	workflow.Status = "compensated"
	c.JSON(200, gin.H{"message": "Workflow compensated"})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}