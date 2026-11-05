package main

import (
	"context"
	"log"
	"net"

	pb "lab3-shared/proto/procesadora"
	processing "lab3-shared/proto/processing"

	"google.golang.org/grpc"
)

type ServidorProcesadora struct {
	pb.UnimplementedProcesadoraServer
	processing.UnimplementedProcesamientoServer

	bully *Bully
	core  *Core
}

func (s *ServidorProcesadora) Eleccion(ctx context.Context, req *pb.EleccionRequest) (*pb.EleccionResponse, error) {
	// recibe la solicitud de eleccion y la deriva al algoritmo bully
	return s.bully.RecibirEleccion(ctx, req)
}

func (s *ServidorProcesadora) Coordinador(ctx context.Context, req *pb.CoordinadorRequest) (*pb.CoordinadorResponse, error) {
	// recibe el anuncio del coordinador y lo deriva al algoritmo bully
	return s.bully.RecibirCoordinador(ctx, req)
}

func (s *ServidorProcesadora) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	// responde que la procesadora sigue disponible
	return &pb.PingResponse{
		Disponible: true,
	}, nil
}

func (s *ServidorProcesadora) RecibirActividad(ctx context.Context, req *processing.ProcesarActividadRequest) (*processing.ProcesarActividadResponse, error) {
	// recibe la actividad del agente y la deriva al core
	return s.core.RecibirActividad(ctx, req)
}

func (s *ServidorProcesadora) ProcesarActividad(ctx context.Context, req *processing.ProcesarActividadRequest) (*processing.ProcesarActividadResponse, error) {
	// recibe la actividad asignada y la deriva al core
	return s.core.ProcesarActividad(ctx, req)
}

func (s *ServidorProcesadora) IniciarServidor(direccion string) error {
	//Inicia el servidor rcp

	listener, err := net.Listen("tcp", direccion)
	if err != nil {
		return err
	}

	servidor := grpc.NewServer()
	pb.RegisterProcesadoraServer(servidor, s)
	processing.RegisterProcesamientoServer(servidor, s)

	go func() {
		log.Printf("[PROCESADORA] Servidor gRPC escuchando en %s", direccion)
		if err := servidor.Serve(listener); err != nil {
			log.Printf("[PROCESADORA] Error en servidor gRPC: %v", err)
		}
	}()

	return nil
}
