package main

import (
	"context"
	"fmt"
	"time"

	pb "lab3-shared/proto/nameserver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	nombreCuartel       = "gamma.almacenamiento.cuartel"
	nombreLider         = "gamma.procesamiento.lider"
	prefijoProcesadoras = "gamma.procesamiento.procesador"
)

type NamingClient struct {
	cliente              pb.NameServerClient
	conexion             *grpc.ClientConn
	nombreProcesadora    string
	direccionProcesadora string
}

// Crea el cliente gRPC del NS
func NewNamingClient(
	direccionNS string,
	id int,
	direccionProcesadora string,
) (*NamingClient, error) {

	conexion, err := grpc.NewClient(
		direccionNS,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	return &NamingClient{
		cliente:              pb.NewNameServerClient(conexion),
		conexion:             conexion,
		nombreProcesadora:    fmt.Sprintf("%s%d", prefijoProcesadoras, id),
		direccionProcesadora: direccionProcesadora,
	}, nil
}

func (n *NamingClient) ResolverCuartel() (string, error) {
	//Resuelve l nombre del caurtel con el NS
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	respuesta, err := n.cliente.ResolverRPC(ctx, &pb.ResolverRequest{
		Nombre: nombreCuartel,
	})
	if err != nil {
		return "", err
	}

	return respuesta.Direccion, nil
}

func (n *NamingClient) RegistrarProcesadora() error {
	//Se registra con el NS
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := n.cliente.RegistrarRPC(ctx, &pb.RegistrarRequest{
		Nombre:    n.nombreProcesadora,
		Direccion: n.direccionProcesadora,
	})

	return err
}

func (n *NamingClient) RenovarProcesadora() error {
	//Renueva el registro con el NS
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := n.cliente.RenovarRPC(ctx, &pb.RenovarRequest{
		Nombre: n.nombreProcesadora,
	})

	return err
}

func (n *NamingClient) RegistrarLider() error {
	//Se registra como lider (cuando lo sea)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := n.cliente.RegistrarRPC(ctx, &pb.RegistrarRequest{
		Nombre:    nombreLider,
		Direccion: n.direccionProcesadora,
	})

	return err
}

func (n *NamingClient) RenovarLider() error {
	//Renueva su registro de lider.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := n.cliente.RenovarRPC(ctx, &pb.RenovarRequest{
		Nombre: nombreLider,
	})

	return err
}

func (n *NamingClient) ListarProcesadoras() ([]*pb.Servicio, error) {
	// Lista las procesadoras registradas y activas (para el bully)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	respuesta, err := n.cliente.ListarRPC(ctx, &pb.ListarRequest{
		Prefijo: prefijoProcesadoras,
	})
	if err != nil {
		return nil, err
	}

	return respuesta.Servicios, nil
}

// Cerrar conexión
func (n *NamingClient) Close() error {
	return n.conexion.Close()
}
