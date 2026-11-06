package main

import (
	"bufio"
	"context"
	"fmt"
	pbCuartel "lab3-shared/proto/cuartel"
	pbNaming "lab3-shared/proto/nameserver"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const nombreCuartel = "gamma.almacenamiento.cuartel"

type NamingClient struct {
	cliente  pbNaming.NameServerClient
	conexion *grpc.ClientConn
}

func NewNamingClient(direccionNS string) (*NamingClient, error) {
	//Igual que en cuartel (y varias más)
	conexion, err := grpc.NewClient(
		direccionNS,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &NamingClient{
		cliente:  pbNaming.NewNameServerClient(conexion),
		conexion: conexion,
	}, nil
}

func (n *NamingClient) ResolverCuartel() (string, error) {
	//Nuevamente similar a cuartel. cambiar que ahora consulta por cuartel
	ctx, cancel := context.WithTimeout(
		context.Background(), 3*time.Second,
	)
	defer cancel()

	respuesta, err := n.cliente.ResolverRPC(ctx, &pbNaming.ResolverRequest{
		Nombre: nombreCuartel,
	})
	if err != nil {
		return "", err
	}

	return respuesta.Direccion, nil
}

func (n *NamingClient) Close() error {
	return n.conexion.Close()
}

func consultarActividades(naming *NamingClient, scanner *bufio.Scanner) {
	//Se encarga sólo de consultar las actividades
	//LA llama obtener activiaddes
	actividades, err := obtenerActividades(naming)
	if err != nil {
		fmt.Printf("[POLICIA] Error consultando actividades: %v\n", err)
		return
	}

	imprimirActividades(actividades)

	fmt.Println("\nPresione ENTER para volver al menú...")
	scanner.Scan()
}

func obtenerActividades(naming *NamingClient) ([]*pbCuartel.Actividad, error) {
	//Conecta con Cuartel y usar el método para obtener las actividades
	for {

		direccion, err := naming.ResolverCuartel()
		if err != nil {
			fmt.Printf("[POLICIA] Error resolviendo Cuartel: %v\n", err)
			time.Sleep(3 * time.Second)
			continue
		}

		conexion, err := grpc.NewClient(direccion, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Printf("[POLICIA] Error conectando con Cuartel: %v\n", err)
			time.Sleep(3 * time.Second)
			continue
		}

		cliente := pbCuartel.NewCuartelClient(conexion)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		respuesta, err := cliente.ConsultarActividades(ctx, &pbCuartel.ConsultarActividadesRequest{})
		cancel()

		conexion.Close()

		if err != nil {
			fmt.Printf("[POLICIA] Error consultando Cuartel: %v\n", err)
			time.Sleep(3 * time.Second)
			continue
		}

		return respuesta.Actividades, nil
	}
}

// Imprime las actividades recibidas.
func imprimirActividades(actividades []*pbCuartel.Actividad) {

	if len(actividades) == 0 {
		fmt.Println("\nNo existen actividades registradas.")
		return
	}

	fmt.Println("\n===================== [RESULTADOS] ==========================")

	for _, actividad := range actividades {

		fmt.Printf("\nID:           %s\n", actividad.ActividadId)
		fmt.Printf("Ubicación:    %s\n", actividad.Ubicacion)
		fmt.Printf("Peligrosidad: %s\n", actividad.Peligrosidad)
		fmt.Printf("Fecha emisión: %s\n", actividad.FechaEmision)

		fmt.Printf("Integrantes:  %s\n",
			strings.Join(actividad.Integrantes, ", "),
		)
	}

	fmt.Println("\n===============================================")
	fmt.Printf("Total de actividades: %d\n", len(actividades))

}

// Interfaz
func mostrarMenu() {
	fmt.Print("\033[H\033[2J")
	fmt.Println("================ [POLICIA] ================")
	fmt.Println()
	fmt.Println("Seleccione una opción:")
	fmt.Println()
	fmt.Println("1. Consultar registros de actividades BD")
	fmt.Println("0. Salir")
	fmt.Println()
	fmt.Print("Opción: ")
}

func consultarOpcion(scanner *bufio.Scanner) int {

	var opcion int

	if !scanner.Scan() {
		return -1
	}

	_, err := fmt.Sscan(scanner.Text(), &opcion)
	if err != nil {
		return -1
	}

	return opcion
}

func main() {

	direccionNS := os.Getenv("NAMESERVER_ADDRESS")

	if direccionNS == "" {
		fmt.Println("[POLICIA] Error: falta la variable NAMESERVER_ADDRESS.")
		return
	}

	naming, err := NewNamingClient(direccionNS)
	if err != nil {
		fmt.Printf("[POLICIA] Error creando cliente Naming: %v\n", err)
		return
	}
	defer naming.Close()

	fmt.Println("[POLICIA] Cliente Naming creado.")

	fmt.Println("[POLICIA] Esperando disponibilidad de Naming y Cuartel...")

	for {
		direccion, err := naming.ResolverCuartel()

		if err != nil || direccion == "" {
			fmt.Println("[POLICIA] Naming o Cuartel todavía no están disponibles. Reintentando en 3 segundos...")
			time.Sleep(3 * time.Second)
			continue
		}

		fmt.Println("[POLICIA] Naming disponible. Cuartel registrado.")
		break
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		mostrarMenu()
		opcion := consultarOpcion(scanner)

		switch opcion {
		case 1:
			consultarActividades(naming, scanner)

		case 0:
			fmt.Println("[POLICIA] Cerrando servicio...")
			return

		case -1:
			fmt.Println("[POLICIA] Entrada inválida o fin de entrada.")
			return

		default:
			fmt.Println("[POLICIA] Opción no válida.")
			fmt.Println("Presione ENTER para continuar...")
			scanner.Scan()
		}
	}
}
