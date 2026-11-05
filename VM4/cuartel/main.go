package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	pb "lab3-shared/proto/cuartel"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Actividad struct {
	ActividadID  string
	Ubicacion    string
	Peligrosidad string
	Integrantes  []string
	FechaEmision time.Time
}

type ServidorCuartel struct {
	pb.UnimplementedCuartelServer
	db *sql.DB
}

func NewServidorCuartel(db *sql.DB) *ServidorCuartel {
	return &ServidorCuartel{
		db: db,
	}
}

// conexion y utilizacion de la bd
func conectarBD() (*sql.DB, error) {
	log.Println("[CUARTEL] Iniciando conexión con MySQL...")

	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	database := os.Getenv("MYSQL_DATABASE")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true",
		user, password, host, port, database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("[CUARTEL] Error abriendo conexión MySQL: %v", err)
		return nil, err
	}

	if err := db.Ping(); err != nil {
		log.Printf("[CUARTEL] MySQL no responde: %v", err)
		db.Close()
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("[CUARTEL] Conexión con MySQL establecida correctamente")

	return db, nil
}

func insertarActividad(db *sql.DB, actividad Actividad) (bool, error) {
	log.Printf(
		"[CUARTEL] Recibida actividad %s. Ubicación: %s | Peligrosidad: %s",
		actividad.ActividadID,
		actividad.Ubicacion,
		actividad.Peligrosidad,
	)

	query := `
        INSERT INTO actividades
        (actividad_id, ubicacion, peligrosidad, integrantes, fecha_emision)
        VALUES (?, ?, ?, ?, ?)
    `

	integrantesJSON, err := json.Marshal(actividad.Integrantes)
	if err != nil {
		log.Printf(
			"[CUARTEL] Error serializando integrantes de actividad %s: %v",
			actividad.ActividadID, err,
		)
		return false, err
	}

	_, err = db.Exec(
		query,
		actividad.ActividadID,
		actividad.Ubicacion,
		actividad.Peligrosidad,
		integrantesJSON,
		actividad.FechaEmision,
	)

	if err != nil {
		var mysqlErr *mysql.MySQLError

		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			log.Printf(
				"[CUARTEL] Actividad %s ya existe. Se ignora el duplicado",
				actividad.ActividadID,
			)
			return false, nil
		}

		log.Printf(
			"[CUARTEL] Error insertando actividad %s: %v",
			actividad.ActividadID, err,
		)
		return false, err
	}

	log.Printf(
		"[CUARTEL] Actividad %s almacenada correctamente",
		actividad.ActividadID,
	)

	return true, nil
}

func obtenerActividades(db *sql.DB) ([]Actividad, error) {
	log.Println("[CUARTEL] Recibida solicitud de consulta de actividades")

	query := `
        SELECT actividad_id, ubicacion, peligrosidad,
               integrantes, fecha_emision
        FROM actividades
    `

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("[CUARTEL] Error consultando actividades: %v", err)
		return nil, err
	}
	defer rows.Close()

	actividades := make([]Actividad, 0)

	for rows.Next() {
		var actividad Actividad
		var integrantesJSON []byte

		err := rows.Scan(
			&actividad.ActividadID,
			&actividad.Ubicacion,
			&actividad.Peligrosidad,
			&integrantesJSON,
			&actividad.FechaEmision,
		)

		if err != nil {
			log.Printf("[CUARTEL] Error leyendo fila de actividad: %v", err)
			return nil, err
		}

		err = json.Unmarshal(integrantesJSON, &actividad.Integrantes)
		if err != nil {
			log.Printf(
				"[CUARTEL] Error deserializando integrantes de actividad %s: %v",
				actividad.ActividadID, err,
			)
			return nil, err
		}

		actividades = append(actividades, actividad)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[CUARTEL] Error recorriendo resultados: %v", err)
		return nil, err
	}

	log.Printf(
		"[CUARTEL] Consulta finalizada. Actividades encontradas: %d",
		len(actividades),
	)

	return actividades, nil
}

func (s *ServidorCuartel) RecibirActividad(
	ctx context.Context,
	req *pb.RecibirActividadRequest,
) (*pb.RecibirActividadResponse, error) {

	//Se encargará de recibir la actividad por la procesadora correspondiente.

	actividadPB := req.Actividad

	log.Printf(
		"[CUARTEL] Procesando actividad %s",
		actividadPB.ActividadId,
	)

	fechaEmision, err := time.Parse(
		"2006-01-02T15:04:05",
		actividadPB.FechaEmision,
	)
	if err != nil {
		log.Printf(
			"[CUARTEL] Fecha inválida en actividad %s: %v",
			actividadPB.ActividadId, err,
		)
		return nil, status.Error(
			codes.InvalidArgument,
			"formato de fecha inválido",
		)
	}
	actividad := Actividad{
		ActividadID:  actividadPB.ActividadId,
		Ubicacion:    actividadPB.Ubicacion,
		Peligrosidad: actividadPB.Peligrosidad,
		Integrantes:  actividadPB.Integrantes,
		FechaEmision: fechaEmision,
	}

	insertada, err := insertarActividad(s.db, actividad)
	if err != nil {
		log.Printf(
			"[CUARTEL] Error almacenando actividad %s: %v",
			actividad.ActividadID, err,
		)
		return nil, status.Error(
			codes.Internal,
			"error almacenando actividad",
		)
	}

	if !insertada {
		log.Printf(
			"[CUARTEL] Actividad %s duplicada. No se insertó nuevamente",
			actividad.ActividadID)

		return &pb.RecibirActividadResponse{
			Recibida: false,
		}, nil
	}

	log.Printf(
		"[CUARTEL] Actividad %s recibida y almacenada correctamente",
		actividad.ActividadID,
	)

	return &pb.RecibirActividadResponse{
		Recibida: true,
	}, nil
}

func (s *ServidorCuartel) ConsultarActividades(
	ctx context.Context,
	req *pb.ConsultarActividadesRequest,
) (*pb.ConsultarActividadesResponse, error) {

	//Consulta las actividades en la BD. Las convierte a protocolbuffers nuevamente
	actividades, err := obtenerActividades(s.db)
	if err != nil {
		log.Printf(
			"[CUARTEL] Error obteniendo actividades: %v",
			err,
		)
		return nil, status.Error(
			codes.Internal,
			"error consultando actividades",
		)
	}

	respuesta := &pb.ConsultarActividadesResponse{
		Actividades: make([]*pb.Actividad, 0, len(actividades)),
	}

	for _, actividad := range actividades {
		respuesta.Actividades = append(
			respuesta.Actividades,
			&pb.Actividad{
				ActividadId:  actividad.ActividadID,
				Ubicacion:    actividad.Ubicacion,
				Peligrosidad: actividad.Peligrosidad,
				Integrantes:  actividad.Integrantes,
				FechaEmision: actividad.FechaEmision.Format(
					"2006-01-02 15:04:05",
				),
			},
		)
	}

	log.Printf(
		"[CUARTEL] Consulta completada. Actividades devueltas: %d",
		len(respuesta.Actividades),
	)

	return respuesta, nil
}

// Inicialización del servidor gRPC
func (s *ServidorCuartel) IniciarServidor(
	direccion string,
	listo chan<- struct{},
) error {

	listener, err := net.Listen("tcp", direccion)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCuartelServer(grpcServer, s)

	close(listo)

	return grpcServer.Serve(listener)
}

/*
  - variables de entorno
  - conexion bd
  - cliente ns
  - registro y renovación ns
  - inicialización  gRPC
    (repeat)
*/
func main() {

	// Leer variables de entorno
	ip := os.Getenv("VM_IP")
	puerto := os.Getenv("CUARTEL_PORT")
	direccionNS := os.Getenv("NAMESERVER_ADDRESS")

	if ip == "" {
		log.Fatal("[CUARTEL] Falta la variable de entorno VM_IP")
	}

	if puerto == "" {
		log.Fatal("[CUARTEL] Falta la variable de entorno CUARTEL_PORT")
	}

	if direccionNS == "" {
		log.Fatal("[CUARTEL] Falta la variable de entorno NAMESERVER_ADDRESS")
	}
	direccionCuartel := fmt.Sprintf("%s:%s", ip, puerto)

	log.Println("[CUARTEL] Iniciando...")

	// Conectarse a MySQL
	var dbErr error
	var db *sql.DB

	for {
		db, dbErr = conectarBD()

		if dbErr != nil {
			log.Printf("[CUARTEL] Error conectando a MySQL: %v. Reintentando en 3 segundos...", dbErr)
			time.Sleep(3 * time.Second)
			continue
		}

		log.Println("[CUARTEL] Conexión a MySQL establecida")
		break
	}
	defer db.Close()

	// Crear cliente del Naming Server
	var naming *NamingClient
	var namingErr error

	for {
		naming, namingErr = NewNamingClient(direccionNS, direccionCuartel)

		if namingErr != nil {
			log.Printf("[CUARTEL] Error creando cliente Naming: %v. Reintentando en 3 segundos...", namingErr)
			time.Sleep(3 * time.Second)
			continue
		}

		log.Println("[CUARTEL] Cliente Naming creado")
		break
	}
	defer naming.Close()

	// Crear servidor gRPC del Cuartel
	servidor := NewServidorCuartel(db)

	// Iniciar servidor gRPC
	servidorListo := make(chan struct{})
	errorServidor := make(chan error, 1)

	go func() {
		errorServidor <- servidor.IniciarServidor(":"+puerto, servidorListo)
	}()

	// Esperar hasta que el servidor esté escuchando
	select {
	case <-servidorListo:
		log.Printf("[CUARTEL] Servidor gRPC escuchando en %s", direccionCuartel)

	case err := <-errorServidor:
		log.Fatalf("[CUARTEL] Error iniciando servidor gRPC: %v", err)
	}

	// Registrar Cuartel en Naming Server
	for {
		if err := naming.RegistrarCuartel(); err != nil {
			log.Printf("[CUARTEL] Error registrándose en Naming Server: %v. Reintentando en 3 segundos...", err)
			time.Sleep(3 * time.Second)
			continue
		}

		log.Println("[CUARTEL] Registrado correctamente en Naming Server")
		break
	}

	// Mantener el registro activo
	go naming.MantenerRegistro()

	// Esperar por si el servidor gRPC termina inesperadamente
	if err := <-errorServidor; err != nil {
		log.Fatalf("[CUARTEL] Error en servidor gRPC: %v", err)
	}
}
