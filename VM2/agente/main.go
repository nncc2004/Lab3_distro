package main
import (
    "fmt"
    "log"
    "os"
    "time"
	"encoding/json"
    amqp "github.com/rabbitmq/amqp091-go"
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


func conectarRabbitMQ() (*amqp.Connection, error){
	//copy paste de la de los distros

	url := os.Getenv("RABBITMQ_URL")

    if url == ""{
        return nil, fmt.Errorf("RABBITMQ_URL no está configurada")
    }

    for{
        conn, err := amqp.Dial(url)

        if err == nil{
            log.Println("[AGENTE] Conectado a RabbitMQ")
            return conn, nil
        }

        log.Printf("[AGENTE] Error al conectar con RabbitMQ: %v", err)
        log.Println("[AGENTE] Reintentando conexión con RabbitMQ en 2 segundos...")

        time.Sleep(2 * time.Second)
    }

}


func leerActividad(mensajes <-chan amqp.Delivery) (Actividad, error){
	//Lee la cola y toma el primer elemento. Sólo 1
    mensaje, ok := <-mensajes

    if !ok{
        return Actividad{}, fmt.Errorf("canal cerrado")
    }

    var actividad Actividad

    err := json.Unmarshal(mensaje.Body, &actividad)

    if err != nil{
        return Actividad{}, fmt.Errorf("error al decodificar actividad: %w", err)
    }

    log.Printf("[AGENTE] Actividad recibida: %s", actividad.ActividadID)

    return actividad, nil
}


func resolverNombreNS(nombre string) (string, error){
	return "", nil
}
func enviarActividadGRPC(direccion string, actividad Actividad) error {
	return nil
}


func main(){

    conn, err := conectarRabbitMQ()
    if err != nil{
        log.Fatal(err)
    }
    defer conn.Close()

    // Crear canal
    canal, err2 := conn.Channel()
    if err2 != nil{
        log.Fatal(err2)
    }
    defer canal.Close()

    // Declarar cola
    cola, err3 := canal.QueueDeclare(
        "actividades",
        true,
        false,
        false,
        false,
        nil,
    )
    if err3 != nil {
        log.Fatal(err3)
    }

    // Iniciar consumidor
    mensajes, err4 := canal.Consume(
        cola.Name,
        "",
        true,
        false,
        false,
        false,
        nil,
    )
    if err4 != nil {
        log.Fatal(err4)
    }

    // Consumir actividades
    for {
        actividad, err5 := leerActividad(mensajes)

        if err5 != nil {
            log.Printf("[AGENTE] Error: %v", err5)
            continue
        }

        log.Printf("[AGENTE] Actividad: %+v", actividad)
    }
}