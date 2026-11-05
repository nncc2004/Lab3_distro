package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "lab3-shared/proto/nameserver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	cliente := pb.NewNameServerClient(conn)

	nombre := "gamma.procesamiento.prueba"
	direccion := "192.168.1.10:50052"

	// Registrar servicio
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	_, err = cliente.RegistrarRPC(ctx, &pb.RegistrarRequest{
		Nombre:    nombre,
		Direccion: direccion,
	})

	cancel()

	if err != nil {
		log.Fatalf("Error al registrar: %v", err)
	}

	fmt.Println("[CLIENTE] Servicio registrado:", nombre)
	fmt.Println("[CLIENTE] Comenzando monitoreo durante 15 segundos")

	// Consultar periódicamente
	inicio := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for time.Since(inicio) < 15*time.Second {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		respuesta, err := cliente.ListarRPC(ctx, &pb.ListarRequest{
			Prefijo: "gamma.procesamiento.",
		})

		cancel()

		if err != nil {
			fmt.Println("[CLIENTE] Error:", err)
		} else if len(respuesta.Servicios) == 0 {
			fmt.Println("[CLIENTE] No hay servicios registrados")
		} else {
			fmt.Println("[CLIENTE] Servicios activos:")

			for _, servicio := range respuesta.Servicios {
				fmt.Printf("  %s -> %s\n", servicio.Nombre, servicio.Direccion)
			}
		}

		<-ticker.C
	}

	fmt.Println("[CLIENTE] Monitoreo finalizado")
}
