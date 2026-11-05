package main

import (
	"context"
	"fmt"
	"time"

	pb "lab3-shared/proto/nameserver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const nombreCuartel = "gamma.almacenamiento.cuartel"

type NamingClient struct {
	cliente   pb.NameServerClient
	conexion  *grpc.ClientConn
	direccion string
}

// Igual que para las procesadoras
func NewNamingClient(
	direccionNS string,
	direccionCuartel string,
) (*NamingClient, error) {

	conexion, err := grpc.NewClient(
		direccionNS,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &NamingClient{
		cliente:   pb.NewNameServerClient(conexion),
		conexion:  conexion,
		direccion: direccionCuartel,
	}, nil
}

func (n *NamingClient) RegistrarCuartel() error {
	//se registra con el NS
	ctx, cancel := context.WithTimeout(
		context.Background(), 3*time.Second,
	)
	defer cancel()

	_, err := n.cliente.RegistrarRPC(ctx, &pb.RegistrarRequest{
		Nombre:    nombreCuartel,
		Direccion: n.direccion,
	})

	return err
}

func (n *NamingClient) RenovarCuartel() error {
	//renueva con el NS
	ctx, cancel := context.WithTimeout(
		context.Background(), 3*time.Second,
	)
	defer cancel()

	_, err := n.cliente.RenovarRPC(ctx, &pb.RenovarRequest{
		Nombre: nombreCuartel,
	})

	return err
}

func (n *NamingClient) MantenerRegistro() {
	//Se encarga de estar renovando, y si falla, se vuelve a registrar
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		err := n.RenovarCuartel()

		if err != nil {
			fmt.Printf("[CUARTEL] Error renovando registro: %v\n", err)

			if err := n.RegistrarCuartel(); err != nil {
				fmt.Printf("[CUARTEL] Error registrando nuevamente: %v\n", err)
			} else {
				fmt.Println("[CUARTEL] Registro recuperado")
			}

			continue
		}

		fmt.Println("[CUARTEL] Registro renovado")
	}
}

func (n *NamingClient) Close() error {
	return n.conexion.Close()
}
