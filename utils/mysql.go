package utils

import (
	"database/sql"
	"fmt"
	"strings"
	"zin-engine/model"

	_ "github.com/go-sql-driver/mysql"
)

var lastSQlConnErr = ""
var mySQL *sql.DB

// ConnectDB uses ENV to create SQL connection and updates mySQL
func ConnectDB(ctx *model.RequestContext) *sql.DB {

	if mySQL != nil {
		return mySQL
	}

	env := ctx.ENV
	host, ok1 := env["MYSQL_HOST"]
	port, ok2 := env["MYSQL_PORT"]
	user, ok3 := env["MYSQL_USER"]
	pass, ok4 := env["MYSQL_PASS"]
	db, ok5 := env["MYSQL_BASE"]

	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		lastSQlConnErr = "Configuration error, check your .env if the mysql db connection details is present"
		return nil
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, pass, host, port, db)
	fmt.Println("SQL Connecting: ", dsn)

	// Connect &Check
	dbConn, err := sql.Open("mysql", dsn)
	if err != nil {
		lastSQlConnErr = fmt.Sprintf("Failed to open connection. Error: %v", err)
		return nil
	}

	pingErr := dbConn.Ping()
	if pingErr != nil {
		lastSQlConnErr = fmt.Sprintf("Failed to communicate with mysql db. Error: %s", pingErr.Error())
		return nil
	}

	lastSQlConnErr = ""
	mySQL = dbConn
	return dbConn
}

func RunQuery(ctx *model.RequestContext, query string, varName string) error {

	// remove unnecessary space from both ends
	query = strings.TrimSpace(query)

	// Check if not a SELECT then no execution will be done
	if !strings.HasPrefix(strings.ToUpper(query), "SELECT") {
		return fmt.Errorf("only SELECT queries are allowed")
	}

	// Check if safe to run
	if !isSafeSQL(query) {
		return fmt.Errorf("dangerous SQL keywords detected. execution revoked")
	}

	// Check & set connection
	if mySQL == nil && ConnectDB(ctx) == nil {
		return fmt.Errorf("sql connection error. %s", lastSQlConnErr)
	}

	// Execute
	result, err := executeQueryAndGetResponse(query)
	if err != nil {
		return err
	}

	// Set response in context
	switch data := result.(type) {
	case []map[string]interface{}:
		list := make([]any, len(data))
		for i, row := range data {
			list[i] = row
		}
		ctx.CustomVar.LIST[varName] = list

	default:
		ctx.CustomVar.Raw[varName] = fmt.Sprintf("%v", data)
	}

	return nil
}

func isSafeSQL(query string) bool {
	unsafeKeywords := []string{
		";", "INSERT", "UPDATE", "DELETE", "DROP", "ALTER",
		"CREATE", "EXEC", "--", "/*", "*/", "XP_",
	}

	upperQuery := strings.ToUpper(query)
	for _, keyword := range unsafeKeywords {
		if strings.Contains(upperQuery, keyword) {
			return false
		}
	}
	return true
}

func executeQueryAndGetResponse(query string) (interface{}, error) {
	rows, err := mySQL.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %s Error: %v", query, err)
	}

	defer rows.Close()

	columns, _ := rows.Columns()
	results := []map[string]interface{}{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		parts := make([]interface{}, len(columns))
		for i := range values {
			parts[i] = &values[i]
		}
		if err := rows.Scan(parts...); err != nil {
			continue
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			if b, ok := values[i].([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = values[i]
			}
		}
		results = append(results, rowMap)
	}

	return results, nil
}
