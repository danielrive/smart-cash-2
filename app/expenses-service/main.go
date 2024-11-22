package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"smart-cash/expenses-service/internal/common"
	"smart-cash/expenses-service/internal/handler"
	"smart-cash/expenses-service/internal/repositories"
	"smart-cash/expenses-service/internal/service"
	"smart-cash/utils"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
)

var logger *slog.Logger

var notToLogEndpoints = []string{"/expenses/health", "/expenses/metrics"}

func main() {
	common.ServiceName = os.Getenv("SERVICE_NAME")
	// Set-up logger handler
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // (Info, Warn, Error)
	}))
	slog.SetDefault(logger)

	// Init OTel TracerProvider
	tp := utils.InitOpenTelemetry(os.Getenv("OTEL_COLLECTOR"), common.ServiceName, logger)

	otel.SetTracerProvider(tp)

	// validate if env variables exists
	expensesTable := os.Getenv("DYNAMODB_EXPENSES_TABLE")
	if expensesTable == "" {
		logger.Error("environment variable not found", slog.String("variable", "DYNAMODB_EXPENSES_TABLE"))
		os.Exit(1)
	}

	awsRegion := os.Getenv("AWS_REGION")

	if awsRegion == "" {
		logger.Error("environment variable not found", slog.String("variable", "AWS_REGION"))
		os.Exit(1)
	}

	// configure the SDK
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		logger.Error("unable to load SDK config", slog.String("error", err.Error()))
	}
	// define uuid helper
	uuidHelper := utils.NewUUIDHelper()

	dynamoClient := dynamodb.NewFromConfig(cfg)
	// create a router with gin
	router := gin.New()

	router.Use(
		otelgin.Middleware(common.ServiceName, otelgin.WithFilter(filterTraces)),
		gin.LoggerWithWriter(gin.DefaultWriter, "/expenses/health"),
		gin.Recovery(), gin.Recovery(),
	)

	// // Initialize expenses repository
	expensesRepo := repositories.NewDynamoDBExpensesRepository(dynamoClient, expensesTable, uuidHelper, logger)

	// Initialize expenses service
	expensesService := service.NewExpensesService(expensesRepo, logger)

	// Init expenses handler
	expensesHandler := handler.NewExpensesHandler(expensesService, logger)

	// create expenses
	router.POST("/expenses/", expensesHandler.CreateExpense)

	// define router for get expenses by tag
	router.GET("/expenses/:expenseId", expensesHandler.GetExpensesById)
	// define router for get expenses by category or userId
	router.GET("/expenses", expensesHandler.GetExpensesByQuery)

	router.POST("/expenses/pay/", expensesHandler.PayExpenses)

	router.DELETE("/expenses/:expenseId", expensesHandler.DeleteExpense)

	// Endpoint to test health check
	router.GET("/expenses/health", expensesHandler.HealthCheck)

	router.Run(":8282")

}

func filterTraces(req *http.Request) bool {
	return slices.Index(notToLogEndpoints, req.URL.Path) == -1
}
