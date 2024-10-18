package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query(`WITH monthly_data AS (
		SELECT
			DATE_TRUNC('month', date) AS dt,
			ROUND(SUM(calcCost), 2) AS totalCost,

			ROUND(SUM(CAST(precipitationMm AS FLOAT)), 2) AS totalPrecipitationMm,
			ROUND(AVG(CAST(precipitationMm AS FLOAT)), 2) AS avgPrecipitationMm,
			ROUND(MAX(CAST(precipitationMm AS FLOAT)), 2) AS maxPrecipitationMm,

			ROUND(AVG(CAST(temperatureCelsius AS FLOAT)), 2) AS avgTemperatureCelsius,
			ROUND(MAX(CAST(temperatureCelsius AS FLOAT)), 2) AS maxTemperatureCelsius,

			ROUND(AVG(CAST(humidityPercent AS FLOAT)), 2) AS avgHumidityPercent,
			ROUND(MAX(CAST(humidityPercent AS FLOAT)), 2) AS maxHumidityPercent,

			ROUND(AVG(CAST(windSpeedMph AS FLOAT)), 2) AS avgWindSpeedMph,
			ROUND(MAX(CAST(windSpeedMph AS FLOAT)), 2) AS maxWindSpeedMph,

			COUNT(DISTINCT DATE_TRUNC('day', date)) AS dataDays,
			EXTRACT(DAYS FROM DATE_TRUNC('month', date) + interval '1 month' - interval '1 day') AS totalDaysInMonth
		FROM
			read_csv_auto('data/*.csv')
		GROUP BY
			DATE_TRUNC('month', date)
	)
	SELECT
		dt,

		totalPrecipitationMm,avgPrecipitationMm,maxPrecipitationMm,
		avgTemperatureCelsius,maxTemperatureCelsius,
		avgHumidityPercent,maxHumidityPercent,
		avgWindSpeedMph,maxWindSpeedMph,

		totalCost,
		ROUND(dataDays / totalDaysInMonth, 2) as percentCoverage,
		ROUND(totalCost * totalDaysInMonth / NULLIF(dataDays, 0), 2) AS estimatedTotalCost,
		LAG(totalCost, 12) OVER (ORDER BY dt) AS previousYearTotalCost
	FROM
		monthly_data
	ORDER BY
		dt DESC;`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("# NV Energy Data")
	fmt.Println("| Date | Total Precipitation (mm) | Avg Precipitation (mm) | Max Precipitation (mm) | Avg Temperature (°C) | Max Temperature (°C) | Avg Humidity (%) | Max Humidity (%) | Avg Wind Speed (mph) | Max Wind Speed (mph) | Total Cost | Percent Coverage | Estimated Total Cost | Previous Year Total Cost |")
	fmt.Println("|:-----|-------------------------:|----------------------:|----------------------:|---------------------:|---------------------:|-----------------:|-----------------:|---------------------:|---------------------:|-----------:|-----------------:|---------------------:|-------------------------:|")

	for rows.Next() {
		var (
			dt                    string
			totalPrecipitationMm  sql.NullFloat64
			avgPrecipitationMm    sql.NullFloat64
			maxPrecipitationMm    sql.NullFloat64
			avgTemperatureCelsius sql.NullFloat64
			maxTemperatureCelsius sql.NullFloat64
			avgHumidityPercent    sql.NullFloat64
			maxHumidityPercent    sql.NullFloat64
			avgWindSpeedMph       sql.NullFloat64
			maxWindSpeedMph       sql.NullFloat64
			totalCost             sql.NullFloat64 // Updated to handle possible NULL values
			percentCoverage       sql.NullFloat64 // Updated to handle possible NULL values
			estimatedTotalCost    sql.NullFloat64 // Updated to handle possible NULL values
			previousYearTotalCost sql.NullFloat64
		)
		if err := rows.Scan(&dt,
			&totalPrecipitationMm, &avgPrecipitationMm, &maxPrecipitationMm,
			&avgTemperatureCelsius, &maxTemperatureCelsius,
			&avgHumidityPercent, &maxHumidityPercent,
			&avgWindSpeedMph, &maxWindSpeedMph,
			&totalCost, &percentCoverage, &estimatedTotalCost, &previousYearTotalCost); err != nil {
			log.Fatal(err)
		}

		formatValue := func(val sql.NullFloat64) string {
			if val.Valid {
				return fmt.Sprintf("%.2f", val.Float64)
			}
			return ""
		}
		parsedTime, err := time.Parse(time.RFC3339, dt)
		if err != nil {
			fmt.Println("Error parsing time:", err)
			return
		}

		fmt.Printf("| %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			parsedTime.Format("2006-01-02"),
			formatValue(totalPrecipitationMm), formatValue(avgPrecipitationMm), formatValue(maxPrecipitationMm),
			formatValue(avgTemperatureCelsius), formatValue(maxTemperatureCelsius),
			formatValue(avgHumidityPercent), formatValue(maxHumidityPercent),
			formatValue(avgWindSpeedMph), formatValue(maxWindSpeedMph),
			formatValue(totalCost), formatValue(percentCoverage), formatValue(estimatedTotalCost), formatValue(previousYearTotalCost))
	}
}
