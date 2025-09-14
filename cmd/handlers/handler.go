package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github/assgn/cmd/config"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	DB *pgxpool.Pool
}

type User struct {
	UserId string `uri:"userId"`
}

type RewardData struct {
	UserId      string    `json:"user_id"`
	StockSymbol string    `json:"stock_symbol"`
	ShareUnit   float64   `json:"units"`
	RewardTime  time.Time `json:"time"`
}

type UserRewardData struct {
	StockSymbol string    `json:"stock_symbol"`
	ShareUnit   float64   `json:"units"`
	RewardTime  time.Time `json:"time"`
}

type TodayRewardStat struct {
	StockSymbol string  `json:"stock_symbol"`
	ShareUnit   float64 `json:"units"`
}

type HistoricalRewardData struct {
	Date           time.Time `json:"data"`
	PortfolioValue float64   `json:"portfolio_value"`
}

type UserPortfolioData struct {
	StockSymbol string  `json:"stock_symbol"`
	ShareUnit   float64 `json:"untis"`
	StockValue  float32 `json:"stock_value"`
}

type BuyRewardData struct {
	UserId      string  `json:"user_id"`
	StockSymbol string  `json:"stock_symbol"`
	ShareUnit   float64 `json:"units"`
}

// add reward
func (s *Server) AddReward(c *gin.Context) {
	var rewardData RewardData
	if err := c.BindJSON(&rewardData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "error parsing data"})
		return
	}
	_, err := s.DB.Exec(context.Background(), "INSERT INTO rewards (user_id, stock_symbol, quantity, reward_time) VALUES ($1,$2,$3,$4)",
		rewardData.UserId, rewardData.StockSymbol, rewardData.ShareUnit, rewardData.RewardTime)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Insert Failed"})
		return
	}

	query := `INSERT INTO user_portfolio (user_id, stock_symbol, quantity) VALUES ($1,$2,$3) ON CONFLICT (user_id, stock_symbol) DO UPDATE SET quantity = user_portfolio.quantity + EXCLUDED.quantity`
	_, err = s.DB.Exec(context.Background(), query, rewardData.UserId, rewardData.StockSymbol, rewardData.ShareUnit)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Insert Failed"})
		return
	}

	c.JSON(http.StatusOK, rewardData)
}

// return todays reward
func (s *Server) TodaysRewards(c *gin.Context) {
	var userId User
	if err := c.ShouldBindUri(&userId); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "error parsing user"})
		return
	}
	rows, err := s.DB.Query(context.Background(), "SELECT stock_symbol,quantity, reward_time  FROM rewards WHERE user_id = $1 AND DATE(reward_time) = CURRENT_DATE", userId.UserId)
	if err != nil {
		return
	}
	defer rows.Close()

	var todaysReward []UserRewardData
	for rows.Next() {
		var rewardData UserRewardData
		rows.Scan(
			&rewardData.StockSymbol,
			&rewardData.ShareUnit,
			&rewardData.RewardTime,
		)
		todaysReward = append(todaysReward, rewardData)
	}

	c.JSON(http.StatusOK, gin.H{"todays_rewards": todaysReward})
}

// return reward history
func (s *Server) RewardHistory(c *gin.Context) {
	var userId User
	if err := c.ShouldBindUri(&userId); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "error parsing user"})
		return
	}
	query := `WITH user_rewards AS (
    SELECT 
        user_id,
        stock_symbol,
        quantity,
        DATE(reward_time) AS reward_date
    FROM public.rewards
    WHERE user_id = $1
),
date_range AS (
    SELECT generate_series(
        (SELECT MIN(reward_date) FROM user_rewards), 
        CURRENT_DATE - 1, 
        interval '1 day'
    )::date AS price_date
),
holdings AS (
    SELECT 
        d.price_date,
        r.stock_symbol,
        SUM(r.quantity) AS total_quantity
    FROM date_range d
    JOIN user_rewards r 
      ON r.reward_date <= d.price_date
    GROUP BY d.price_date, r.stock_symbol
),
portfolio_value AS (
    SELECT 
        h.price_date,
        SUM(h.total_quantity * sp.stock_price) AS portfolio_value
    FROM holdings h
    JOIN stock_prices sp 
      ON sp.stock_symbol = h.stock_symbol 
     AND sp.price_date = h.price_date
    GROUP BY h.price_date
    ORDER BY h.price_date
)
SELECT * FROM portfolio_value;
`
	rows, err := s.DB.Query(context.Background(), query, userId.UserId)
	if err != nil {
		return
	}
	defer rows.Close()

	var rewardHistoryValue []HistoricalRewardData
	for rows.Next() {
		var rewardValueData HistoricalRewardData
		rows.Scan(

			&rewardValueData.Date,
			&rewardValueData.PortfolioValue,
		)

		rewardHistoryValue = append(rewardHistoryValue, rewardValueData)
	}

	c.JSON(http.StatusOK, gin.H{"portfolio_reward_value_history": rewardHistoryValue})
}

// return todays reward status
func (s *Server) TodaysRewardsStats(c *gin.Context) {
	var userId User
	if err := c.ShouldBindUri(&userId); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "error parsing user"})
		return
	}
	query := `
SELECT r.stock_symbol, 
       SUM(r.quantity) AS total_quantity      
FROM public.rewards r
JOIN (
    SELECT DISTINCT ON (stock_symbol) stock_symbol, stock_price 
    FROM stock_prices 
    ORDER BY stock_symbol, price_date desc
) sp ON LOWER(r.stock_symbol) = LOWER(sp.stock_symbol)
WHERE r.user_id = $1
  AND DATE(r.reward_time) = CURRENT_DATE
GROUP BY r.user_id, r.stock_symbol, sp.stock_price
`

	rows, err := s.DB.Query(context.Background(), query, userId.UserId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "error parsing data"})
		return
	}

	defer rows.Close()

	var todayRewardStat []TodayRewardStat

	for rows.Next() {
		var rewardStat TodayRewardStat
		rows.Scan(
			&rewardStat.StockSymbol,
			&rewardStat.ShareUnit,
		)
		todayRewardStat = append(todayRewardStat, rewardStat)
	}

	query = `SELECT p.quantity * sp.stock_price  AS stock_value
FROM public.user_portfolio p JOIN (
SELECT DISTINCT ON (stock_symbol) stock_symbol, stock_price FROM stock_prices
ORDER BY stock_symbol, price_date desc
) sp ON LOWER(p.stock_symbol) = LOWER(sp.stock_symbol)
WHERE p.user_id = $1`

	rows, err = s.DB.Query(context.Background(), query, userId.UserId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "error parsing data"})
		return
	}

	defer rows.Close()

	var totalReward float64
	for rows.Next() {
		var rewardValue float64
		rows.Scan(
			&totalReward,
		)
		totalReward += rewardValue
	}

	c.JSON(http.StatusOK, gin.H{"portfolio_value": totalReward, "today_reward_stat": todayRewardStat})
}

// return user portfolio stats
func (s *Server) UserPortfolio(c *gin.Context) {
	var userId User
	if err := c.ShouldBindUri(&userId); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "error parsing user"})
		return
	}

	query := `SELECT p.stock_symbol, p.quantity, p.quantity * sp.stock_price  AS stock_value
FROM public.user_portfolio p JOIN (
SELECT DISTINCT ON (stock_symbol) stock_symbol, stock_price FROM stock_prices
ORDER BY stock_symbol, price_date desc
) sp ON LOWER(p.stock_symbol) = LOWER(sp.stock_symbol)
WHERE p.user_id = $1`

	rows, err := s.DB.Query(context.Background(), query, userId.UserId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "error parsing data"})
		return
	}

	defer rows.Close()

	var UserPortfolio []UserPortfolioData
	for rows.Next() {
		var userPortfolio UserPortfolioData
		rows.Scan(
			&userPortfolio.StockSymbol,
			&userPortfolio.ShareUnit,
			&userPortfolio.StockValue,
		)
		UserPortfolio = append(UserPortfolio, userPortfolio)
	}

	c.JSON(http.StatusOK, gin.H{"portfolio_data": UserPortfolio})
}




//buy stock, call add reward api and update the ledger
func (s *Server) BuyReward(c *gin.Context) {

	var buyReward BuyRewardData
	if err := c.BindJSON(&buyReward); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	//request to nse/bse to buy stock
	var stockPrice float64
	query := `SELECT stock_price FROM stock_prices WHERE stock_symbol = $1 AND DATE(price_date) = CURRENT_DATE`
	err := s.DB.QueryRow(context.Background(), query, buyReward.StockSymbol).Scan(&stockPrice)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "error fetching price"})
		return
	}

	stocksValue := buyReward.ShareUnit * stockPrice

	fees := stocksValue * 0.05

	finalValue := stocksValue + fees

	//update company ledger
	query = `INSERT INTO stock_ledger (stock_symbol, quantity, transaction_type, amount, fees) VALUES ($1, $2, $3, $4, $5)`

	_, err = s.DB.Exec(context.Background(), query,
		buyReward.StockSymbol,
		buyReward.ShareUnit,
		"debit",
		finalValue,
		fees,
	)

	if err != nil {
		panic(err)
	}


	uri := fmt.Sprintf("%s/reward", config.AppConfig.URL)

	now := time.Now()
	formattedTime := now.Format("2006-01-02T15:04:05Z07:00")
	parsedTime, _ := time.Parse("2006-01-02T15:04:05Z07:00", formattedTime)

	rewardData := RewardData{
		UserId:      buyReward.UserId,
		StockSymbol: buyReward.StockSymbol,
		ShareUnit:   buyReward.ShareUnit,
		RewardTime:  parsedTime,
	}

	jsonData, err := json.Marshal(rewardData)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(uri, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var result RewardData

	json.NewDecoder(resp.Body).Decode(&result)

	c.JSON(http.StatusOK, gin.H{"stocks purchased and reward credited": result})

}
