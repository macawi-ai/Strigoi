package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	
	_ "github.com/marcboeker/go-duckdb"
)

func main() {
	// Open database
	db, err := sql.Open("duckdb", "test_json.duckdb")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()
	
	// Create test table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS test_json (
			id INTEGER PRIMARY KEY,
			data JSON
		)
	`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
	
	// Test 1: Insert JSON as string
	testData := map[string]interface{}{
		"name": "test",
		"value": 42,
		"nested": map[string]interface{}{
			"key": "value",
		},
	}
	
	jsonBytes, _ := json.Marshal(testData)
	jsonStr := string(jsonBytes)
	
	fmt.Println("Inserting JSON:", jsonStr)
	
	_, err = db.Exec("INSERT INTO test_json (id, data) VALUES (?, ?)", 1, jsonStr)
	if err != nil {
		log.Fatal("Failed to insert:", err)
	}
	
	// Test 2: Query JSON
	var result sql.NullString
	err = db.QueryRow("SELECT data FROM test_json WHERE id = ?", 1).Scan(&result)
	if err != nil {
		log.Fatal("Failed to query:", err)
	}
	
	fmt.Println("Retrieved JSON:", result.String)
	
	// Test 3: Check what DuckDB returns for JSON columns
	rows, err := db.Query("SELECT data, typeof(data) FROM test_json")
	if err != nil {
		log.Fatal("Failed to query type:", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var data interface{}
		var dataType string
		err := rows.Scan(&data, &dataType)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		fmt.Printf("Data type from DuckDB: %s, Go type: %T\n", dataType, data)
	}
}