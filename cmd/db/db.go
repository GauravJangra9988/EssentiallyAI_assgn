package db

import (
	"context"
	"fmt"
	"github/assgn/cmd/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func DBConnect() (*pgxpool.Pool, error) {
	db_link := fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable", 
	config.AppConfig.DB_Username,
	config.AppConfig.DB_Password,
	config.AppConfig.DB_Name,

)
	conn, err := pgxpool.New(context.Background(),db_link); 
	if err != nil{
		return nil, err
}

	fmt.Println("Connected to database")

	return conn, nil

}
