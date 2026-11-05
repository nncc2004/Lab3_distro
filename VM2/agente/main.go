package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	ns "lab3-shared/proto/nameserver"
	pb "lab3-shared/proto/processing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

/*
1. Leer cola rabbitMQ y extrae actividad
2. Resuelve nombre líder en nameserver
3. Envía la actividad por gRPC al lider
repeat
*/

type Actividad struct {
	ActividadID  string   `json:"actividad_id"`
	Ubicacion    string   `json:"ubicacion"`
	Peligrosidad string   `json:"peligrosidad"`
	Integrantes  []string `json:"integrantes"`
	FechaEmision string   `json:"fecha_emision"`
}

func conectarRabbitMQ() (*amqp.Connection, error) {
	//copy paste de la de los distros

	url := os.Getenv("RABBITMQ_URL")

	if url == "" {
		return nil, fmt.Errorf("RABBITMQ_URL no está configurada")
	}

	for {
		conn, err := amqp.Dial(url)

		if err == nil {
			log.Println("[AGENTE] Conectado a RabbitMQ")
			return conn, nil
		}

		log.Printf("[AGENTE] Error al conectar con RabbitMQ: %v", err)
		log.Println("[AGENTE] Reintentando conexión con RabbitMQ en 2 segundos...")

		time.Sleep(2 * time.Second)
	}

}

func leerActividad(mensajes <-chan amqp.Delivery) (Actividad, error) {
	//Lee la cola y toma el primer elemento. Sólo 1
	mensaje, ok := <-mensajes

	if !ok {
		return Actividad{}, fmt.Errorf("canal cerrado")
	}

	var actividad Actividad

	err := json.Unmarshal(mensaje.Body, &actividad)

	if err != nil {
		return Actividad{}, fmt.Errorf("error al decodificar actividad: %w", err)
	}

	log.Printf("[AGENTE] Actividad recibida: %s", actividad.ActividadID)

	return actividad, nil
}

// DEsde aquí viene lo del NS y gRPC
func conectarNameServer() (ns.NameServerClient, *grpc.ClientConn, error) {
	//Inicia el gRPC y se conecta al nameserver
	direccion := os.Getenv("NAMESERVER_ADDRESS")

	conn, err := grpc.NewClient(
		direccion,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("error conectando con NameServer: %w", err)
	}

	cliente := ns.NewNameServerClient(conn)

	log.Println("[AGENTE] Cliente NameServer inicializado")

	return cliente, conn, nil
}

func resolverNombreNS(cliente ns.NameServerClient, nombre string) (string, error) {
	//pide el nombre del lider en el NS
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	respuesta, err := cliente.ResolverRPC(ctx, &ns.ResolverRequest{
		Nombre: nombre,
	})

	if err != nil {
		return "", fmt.Errorf("error resolviendo %s: %w", nombre, err)
	}

	if respuesta.Direccion == "" {
		return "", fmt.Errorf("el NameServer devolvió una dirección vacía")
	}

	log.Printf("[AGENTE] Nombre resuelto: %s -> %s", nombre, respuesta.Direccion)

	return respuesta.Direccion, nil
}

func enviarActividadGRPC(direccion string, actividad Actividad) error {
	//Intenta enviar la actividad a una dirección
	conn, err := grpc.NewClient(
		direccion,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return fmt.Errorf("error conectando con líder: %w", err)
	}

	defer conn.Close()

	cliente := pb.NewProcesamientoClient(conn)

	// Convertir la actividad local a protobuf.
	solicitud := &pb.ProcesarActividadRequest{
		Actividad: &pb.Actividad{
			ActividadId:  actividad.ActividadID,
			Ubicacion:    actividad.Ubicacion,
			Peligrosidad: actividad.Peligrosidad,
			Integrantes:  actividad.Integrantes,
			FechaEmision: actividad.FechaEmision,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	respuesta, err := cliente.RecibirActividad(ctx, solicitud)

	if err != nil {
		return fmt.Errorf("error enviando actividad al líder: %w", err)
	}

	if !respuesta.Exito {
		return fmt.Errorf("el líder rechazó la actividad: %s", respuesta.Mensaje)
	}

	log.Printf("[AGENTE] Actividad %s entregada al líder: %s",
		actividad.ActividadID, respuesta.Mensaje)

	return nil
}

func entregarActividad(cliente ns.NameServerClient, actividad Actividad) error {

	//Consulta naming y envía la actividad. Reintenta en caso de fallo tras 2s

	nombreLider := "gamma.procesamiento.lider"

	for {
		// Resolver nuevamente la dirección del líder.
		direccion, err := resolverNombreNS(cliente, nombreLider)

		if err != nil {
			log.Printf("[AGENTE] Error resolviendo líder: %v", err)
			log.Println("[AGENTE] Reintentando en 2 segundos...")
			time.Sleep(2 * time.Second)
			continue
		}

		// Enviar actividad al líder.
		err = enviarActividadGRPC(direccion, actividad)

		if err != nil {
			log.Printf("[AGENTE] Error enviando actividad: %v", err)
			log.Println("[AGENTE] Volviendo a resolver líder...")
			time.Sleep(2 * time.Second)
			continue
		}

		log.Printf("[AGENTE] Actividad %s entregada correctamente", actividad.ActividadID)

		return nil
	}
}

func main() {
	// conectar  NS
	naming, connNS, err := conectarNameServer()
	if err != nil {
		log.Fatal(err)
	}
	defer connNS.Close()

	// conectarRabbitMQ
	conn, err := conectarRabbitMQ()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	canal, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer canal.Close()

	// cola rabbit
	cola, err := canal.QueueDeclare(
		"actividades",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	// leer actovidades
	mensajes, err := canal.Consume(
		cola.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("[AGENTE] Esperando actividades...")

	// entregar actividades
	for {
		actividad, err := leerActividad(mensajes)
		if err != nil {
			log.Printf("[AGENTE] Error al recibir actividad: %v", err)
			continue
		}

		log.Printf(
			"[AGENTE] Procesando actividad recibida: %s",
			actividad.ActividadID,
		)

		entregarActividad(naming, actividad)
	}
}

// func main() {

// 	conn, err := conectarRabbitMQ()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer conn.Close()

// 	// Crear canal
// 	canal, err2 := conn.Channel()
// 	if err2 != nil {
// 		log.Fatal(err2)
// 	}
// 	defer canal.Close()

// 	// Declarar cola
// 	cola, err3 := canal.QueueDeclare(
// 		"actividades",
// 		true,
// 		false,
// 		false,
// 		false,
// 		nil,
// 	)
// 	if err3 != nil {
// 		log.Fatal(err3)
// 	}

// 	// Iniciar consumidor
// 	mensajes, err4 := canal.Consume(
// 		cola.Name,
// 		"",
// 		true,
// 		false,
// 		false,
// 		false,
// 		nil,
// 	)
// 	if err4 != nil {
// 		log.Fatal(err4)
// 	}

// 	// Consumir actividades
// 	for {
// 		actividad, err5 := leerActividad(mensajes)

// 		if err5 != nil {
// 			log.Printf("[AGENTE] Error: %v", err5)
// 			continue
// 		}

// 		log.Printf("[AGENTE] Actividad: %+v", actividad)
// 	}
// }
