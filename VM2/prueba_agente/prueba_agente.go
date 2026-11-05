package main

import (
	"context"
	"log"
	"os"
	"time"

	ns "lab3-shared/proto/nameserver"
	pb "lab3-shared/proto/processing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const nombreLider = "gamma.procesamiento.lider"

func main() {
	direccionNS := os.Getenv("NAMESERVER_ADDRESS")

	if direccionNS == "" {
		log.Fatal("NAMESERVER_ADDRESS no configurada")
	}

	// Crea el cliente gRPC del NameServer.
	connNS, err := grpc.NewClient(
		direccionNS,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Error conectando con NameServer: %v", err)
	}
	defer connNS.Close()

	naming := ns.NewNameServerClient(connNS)

	// Actividad de prueba.
	actividad := &pb.ProcesarActividadRequest{
		Actividad: &pb.Actividad{
			ActividadId:  "actividad-prueba-001",
			Ubicacion:    "Santiago",
			Peligrosidad: "ALTA",
			Integrantes:  []string{"Disto 1", "Disto 2"},
			FechaEmision: "2026-11-05",
		},
	}

	log.Println("[AGENTE-PRUEBA] Iniciando envío de actividades")

	for {
		// Resuelve la direccion del lider mediante el NameServer.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		respuestaNS, err := naming.ResolverRPC(ctx, &ns.ResolverRequest{
			Nombre: nombreLider,
		})
		cancel()

		if err != nil {
			log.Printf("[AGENTE-PRUEBA] Error resolviendo lider: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Conecta con el lider.
		connLider, err := grpc.NewClient(
			respuestaNS.Direccion,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			log.Printf("[AGENTE-PRUEBA] Error conectando con lider: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		cliente := pb.NewProcesamientoClient(connLider)

		// Envia la actividad al lider.
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

		respuesta, err := cliente.RecibirActividad(ctx, actividad)

		cancel()
		connLider.Close()

		if err != nil {
			log.Printf("[AGENTE-PRUEBA] Error enviando actividad: %v", err)
		} else {
			log.Printf("[AGENTE-PRUEBA] Respuesta del lider: Exito=%t, Mensaje=%s",
				respuesta.Exito, respuesta.Mensaje)
		}

		time.Sleep(5 * time.Second)
	}
}
