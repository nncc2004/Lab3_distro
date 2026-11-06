package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Actividad struct {
	ActividadID  string   `json:"actividad_id"`
	Ubicacion    string   `json:"ubicacion"`
	Peligrosidad string   `json:"peligrosidad"`
	Integrantes  []string `json:"integrantes"`
	FechaEmision string   `json:"fecha_emision"`
}

func conectarRabbitMQ() (*amqp.Connection, error) {
	/* Se encarga de establecer la conexión con rabbit
	Si falla, espera y vuelve a intentarlo, no termina
	hasta lograr la conexión.
	*/
	url := os.Getenv("RABBITMQ_URL")

	if url == "" {
		return nil, fmt.Errorf("RABBITMQ_URL no está configurada")
	}

	for {
		conn, err := amqp.Dial(url)

		if err == nil {
			log.Println("[LOS DISTROS] Conectado a RabbitMQ")
			return conn, nil
		}

		log.Printf("[LOS DISTROS] Error al conectar con RabbitMQ: %v", err)
		log.Println("[LOS DISTROS] Reintentando conexión con RabbitMQ en 2 segundos...")

		time.Sleep(2 * time.Second)
	}
}

func leerActividad(archivo *csv.Reader) (Actividad, error) {
	/*
		Se encarga de leer la siguiente línea y
		retornar la actividad, o avisar del fin del archivo.
		Lo retorna en el formato  de arhcivo
	*/

	registro, err := archivo.Read()

	if err != nil {
		return Actividad{}, err
	}

	if len(registro) != 5 {
		return Actividad{}, fmt.Errorf("Registro en formato inválido. No son 5 columnas", len(registro))
	}

	if registro[0] == "actividad_id" {
		log.Println("[LOS DISTROS] Cabecera CSV detectada, omitiendo fila")
		return leerActividad(archivo)
	}
	actividad := Actividad{
		ActividadID:  registro[0],
		Ubicacion:    registro[1],
		Peligrosidad: registro[2],
		Integrantes:  strings.Split(registro[3], ","),
		FechaEmision: registro[4],
	}

	log.Printf("[LOS DISTROS] Actividad extraída: %s || %s ||\n", actividad.ActividadID, actividad.FechaEmision)
	return actividad, nil
}

func publicarActividad(actividad Actividad, canal *amqp.Channel) (bool, error) {
	//Se encarga de publicar la actividad en la cola de rabbitMQ
	//La idea es que retorne éxito / fracaso
	//En caso de fracaso, desde main se vuelve a intentar la conexion con rabbit

	mensaje, err := json.Marshal(actividad)
	if err != nil {
		return false, fmt.Errorf("error al convertir actividad a JSON: %w", err)
	}

	err = canal.Publish(
		"",
		"actividades",
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         mensaje,
		},
	)

	if err != nil {
		return false, fmt.Errorf("error al publicar actividad %s: %w", actividad.ActividadID, err)
	}

	log.Printf("[LOS DISTROS] Actividad %s publicada en RabbitMQ", actividad.ActividadID)

	return true, nil

}

func esperarAgente(canal *amqp.Channel, cola amqp.Queue) {
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
		log.Fatalf("[LOS DISTROS] Error al esperar al Agente: %v", err)
	}

	log.Println("[LOS DISTROS] Esperando confirmación del Agente...")

	mensaje, ok := <-mensajes
	if !ok {
		log.Fatal("[LOS DISTROS] Canal de confirmación cerrado")
	}

	log.Printf("[LOS DISTROS] Confirmación recibida: %s", string(mensaje.Body))
	log.Println("[LOS DISTROS] Agente disponible. Comenzando publicación.")
}

/*
1. Abrir y leer el csv. Retornar un 'JSON' con la información
2. Publica en la cola de rabbitmQ
*/

func main() {
	// Abrir el csv
	nombreArchivo := os.Getenv("ARCHIVO_CSV")
	archivoCSV, err1 := os.Open("csv/" + nombreArchivo)
	if err1 != nil {
		log.Fatalf("[LOS DISTROS] Error al abrir CSV: %v", err1)
	}
	defer archivoCSV.Close()

	archivo := csv.NewReader(archivoCSV)
	archivo.Comma = ';'

	// Establecer conexión con rabbit
	conn, _ := conectarRabbitMQ()
	canal, err3 := conn.Channel()
	if err3 != nil {
		log.Fatalf("[LOS DISTROS] Error al crear canal: %v", err3)
	}
	defer canal.Close()

	colaReady, err := canal.QueueDeclare(
		"gamma.ready",
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)

	if err != nil {
		log.Fatalf("[LOS DISTROS] Error al declarar cola de disponibilidad: %v", err)
	}
	esperarAgente(canal, colaReady)
	// Leer línea archivo
	act, err4 := leerActividad(archivo)
	for {
		// Aqui va el comprobador de si es última línea según el retorno en leerActividad
		if err4 == io.EOF {
			log.Println("[LOS DISTROS] No hay más actividades.. Durmiendo por 1 minuto y chequeando de nuevo")
			time.Sleep(time.Minute)
			act, err4 = leerActividad(archivo)
			continue
		}

		if err4 != nil {
			log.Fatalf("[LOS DISTROS] Error al leer CSV: %v", err4)
		}

		_, err5 := publicarActividad(act, canal)

		if err5 == nil {
			act, err4 = leerActividad(archivo)
		} else {
			log.Printf("[LOS DISTROS] Error al publicar: %v", err5)
		}
	}
}
