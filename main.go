package main

import (
	"fmt"
	"github/assgn/cmd/config"
	"github/assgn/cmd/db"
	handler "github/assgn/cmd/handlers"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {

	config.LoadConfig()
	conn, _ := db.DBConnect()
	defer conn.Close()

	now := time.Now()
	formatted := now.Format("2006-01-02T15:04:05Z07:00")
	fmt.Println(formatted)

	server := &handler.Server{DB: conn}

	router := gin.Default()

	router.POST("/buy_reward", server.BuyReward)

	router.POST("/reward", server.AddReward)
	router.GET("/today-stocks/:userId", server.TodaysRewards)
	router.GET("/historical-inr/:userId", server.RewardHistory)
	router.GET("/stats/:userId", server.TodaysRewardsStats)
	router.GET("/portfolio/:userId", server.UserPortfolio)

	router.Run()

}
