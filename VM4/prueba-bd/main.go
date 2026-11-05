package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {

	log.Println("[PRUEBA-BD] Iniciando...")

	// Configuración de conexión
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	database := os.Getenv("MYSQL_DATABASE")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true",
		user, password, host, port, database,
	)

	// Conectar a MySQL
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("[PRUEBA-BD] Error al abrir conexión: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("[PRUEBA-BD] Error de conexión: ", err)
	}

	log.Println("[PRUEBA-BD] Conexión establecida")

	// Insertar actividad
	queryInsert := `
		INSERT INTO actividades
		(actividad_id, ubicacion, peligrosidad, integrantes, fecha_emision)
		VALUES (?, ?, ?, ?, ?)
	`

	integrantes := `["Nicolas", "Vale"]`

	_, err = db.Exec(
		queryInsert,
		"TEST-001",
		"Puerto",
		"Alta",
		integrantes,
		"2026-11-05 20:00:00",
	)

	if err != nil {
		log.Fatal("[PRUEBA-BD] Error al insertar: ", err)
	}

	log.Println("[PRUEBA-BD] Actividad insertada correctamente")

	// Consultar actividades
	querySelect := `
		SELECT actividad_id, ubicacion, peligrosidad,
		       integrantes, fecha_emision
		FROM actividades
	`

	rows, err := db.Query(querySelect)
	if err != nil {
		log.Fatal("[PRUEBA-BD] Error en consulta: ", err)
	}
	defer rows.Close()

	log.Println("[PRUEBA-BD] Resultados:")

	for rows.Next() {

		var id, ubicacion, peligrosidad string
		var integrantes []byte
		var fecha string

		err := rows.Scan(
			&id,
			&ubicacion,
			&peligrosidad,
			&integrantes,
			&fecha,
		)

		if err != nil {
			log.Fatal("[PRUEBA-BD] Error al leer fila: ", err)
		}

		log.Printf(
			"%s | %s | %s | %s | %s",
			id,
			ubicacion,
			peligrosidad,
			string(integrantes),
			fecha,
		)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("[PRUEBA-BD] Error recorriendo resultados: ", err)
	}

	log.Println("[PRUEBA-BD] Prueba finalizada")
}
