package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

type FlexDBTerminal struct {
	db *sql.DB
}

func NewFlexDBTerminal(dbPath string) (*FlexDBTerminal, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Connect to database failed: %v", err)
	}
	return &FlexDBTerminal{db: db}, nil
}

func (t *FlexDBTerminal) ExecuteQuery(sqlString string) error {
	sqlString = strings.TrimSpace(sqlString)
	
	rows, err := t.db.Query(sqlString)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return nil
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	fmt.Println(strings.Join(columns, " | "))
	fmt.Println(strings.Repeat("-", len(strings.Join(columns, " | "))))

	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return err
		}
		
		rowResult := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				rowResult[i] = "NULL"
			} else {
				rowResult[i] = fmt.Sprintf("%v", val)
			}
		}
		fmt.Println(strings.Join(rowResult, " | "))
	}

	return nil
}

func (t *FlexDBTerminal) Close() {
	t.db.Close()
}

func ensureStarListTable(t *FlexDBTerminal) error {
	_, err := t.db.Exec(`
		CREATE TABLE IF NOT EXISTS starList (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL
		)
	`)
	return err
}

func readLastSQL() (string, error) {
	content, err := os.ReadFile("./DBTerminal/lastSql.txt")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func writeLastSQL(sql string) error {
	// TODO: create directory manually now.
	// err := os.MkdirAll("./DBTerminal", 0755)
	// if err != nil {
	// 	return err
	// }
	return os.WriteFile("./DBTerminal/lastSql.txt", []byte(sql), 0644)
}

func readDefaultPath() (string, error) {
	content, err := os.ReadFile("./DBTerminal/defaultPath.txt")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func writeDefaultPath(path string) error {
	return os.WriteFile("./DBTerminal/defaultPath.txt", []byte(path), 0644)
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	dbPathFlag := flag.String("db", "", "databasepath")
	dbPath := *dbPathFlag
	flag.Parse()
	if dbPath == "" {
		path, err := readDefaultPath()
		dbPath = path
		if err != nil {
			// user input path
			fmt.Print("enter sql path: ")
			if !scanner.Scan() {
				log.Fatal(err)
			}
			
			dbPath = scanner.Text()
			writeDefaultPath(dbPath)
		}
	}

	terminal, err := NewFlexDBTerminal(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer terminal.Close()

	// Open the star list database
	startDbPath := flag.String("stardb", "./DBTerminal/dbterminal.db", "databasepath")
	flag.Parse()
	starDB, err := NewFlexDBTerminal(*startDbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer starDB.Close()

	// Ensure the starList table exists
	if err := ensureStarListTable(starDB); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Connect to database: %s\n", dbPath)
	fmt.Println("In interactive mode now (Type 'exit' to leave)")

	sqlBuffer := ""

	for {
		fmt.Print("SQL> ")
		if !scanner.Scan() {
			break
		}
		
		line := scanner.Text()
		
		if strings.ToLower(line) == "exit" {
			break
		} else if strings.ToLower(line) == "ls" {
			rows, err := starDB.db.Query("SELECT id, content FROM starList ORDER BY id")
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			
			hasData := false
			for rows.Next() {
				hasData = true
				var id int
				var content string
				if err := rows.Scan(&id, &content); err != nil {
					fmt.Println("Error:", err)
					continue
				}
				fmt.Printf("%d. %s\n", id, content)
			}
			rows.Close()
			
			if !hasData {
				fmt.Println("no data")
			}
			continue
		} else if strings.ToLower(line) == "unstar" {
			lastSQL, err := readLastSQL()
			if err != nil {
				fmt.Println("Error reading last SQL:", err)
				continue
			}
			
			_, err = starDB.db.Exec("DELETE FROM starList WHERE content = ?", lastSQL)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			
			fmt.Println("success")
			continue
		} else if strings.ToLower(line) == "star" {
			lastSQL, err := readLastSQL()
			if err != nil {
				fmt.Println("Error reading last SQL:", err)
				continue
			}
			
			_, err = starDB.db.Exec("INSERT INTO starList (content) VALUES (?)", lastSQL)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			
			fmt.Println("starred")
			continue
		} else if strings.ToLower(line) == "zip" {
			_, err = starDB.db.Exec(`
				WITH numbered_rows AS (
					SELECT id, content, ROW_NUMBER() OVER (ORDER BY id) as new_id 
					FROM starList
				)
				UPDATE starList
				SET id = (
					SELECT new_id 
					FROM numbered_rows 
					WHERE numbered_rows.id = starList.id
				);
				-- 重置自增計數器

				DELETE FROM sqlite_sequence WHERE name = 'starList';`)

			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			fmt.Println("zipped")
			continue

		} else if id, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			var content string
			err := starDB.db.QueryRow("SELECT content FROM starList WHERE id = ?", id).Scan(&content)
			if err == sql.ErrNoRows {
				fmt.Println("no such index.")
				continue
			} else if err != nil {
				fmt.Println("Error:", err)
				continue
			}
			
			fmt.Printf("SQL> %s\n", content)
			sqlBuffer = content
		} else {
			sqlBuffer += line + " "
		}
		
		if strings.HasSuffix(strings.TrimSpace(sqlBuffer), ";") {
			err = terminal.ExecuteQuery(strings.TrimSpace(sqlBuffer))
			if err != nil {
				fmt.Println("Execute error:", err)
			}

			if err := writeLastSQL(strings.TrimSpace(sqlBuffer)); err != nil {
				fmt.Println("Error saving last SQL:", err)
			}

			sqlBuffer = ""
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}