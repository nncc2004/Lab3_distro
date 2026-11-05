// RR: P1 - P2 - P3
package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	ns "lab3-shared/proto/nameserver"
	pb "lab3-shared/proto/processing"

	cuartelpb "lab3-shared/proto/cuartel"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const prefijoProcesadorasCore = "gamma.procesamiento.procesador"

type Core struct {
	pb.UnimplementedProcesamientoServer

	ID     int
	Bully  *Bully
	Naming *NamingClient

	ultimoAsignado int
	mu             sync.Mutex
}

func (c *Core) RecibirActividad(
	ctx context.Context,
	req *pb.ProcesarActividadRequest,
) (*pb.ProcesarActividadResponse, error) {
	// Recibe una actividad del Agente y la distribuye entre las procesadoras activas.

	// Verifica que esta procesadora sea el lider
	c.Bully.mu.Lock()
	esLider := c.Bully.LiderID == c.ID
	c.Bully.mu.Unlock()

	if !esLider {
		return &pb.ProcesarActividadResponse{
			Exito:   false,
			Mensaje: "Esta procesadora no es el lider",
		}, nil
	}

	if req == nil || req.Actividad == nil {
		return &pb.ProcesarActividadResponse{
			Exito:   false,
			Mensaje: "La actividad recibida es invalida",
		}, nil
	}

	log.Printf("[PROCESADORA-%d] Actividad recibida del Agente: ID=%s",
		c.ID, req.Actividad.ActividadId)

	// Consulta las procesadoras que siguen activas
	procesadoras, err := c.ObtenerProcesadorasActivas()
	if err != nil {
		log.Printf("[PROCESADORA-%d] Error obteniendo procesadoras activas: %v", c.ID, err)
		return &pb.ProcesarActividadResponse{
			Exito:   false,
			Mensaje: "Error obteniendo procesadoras activas",
		}, err
	}

	if len(procesadoras) == 0 {
		log.Printf("[PROCESADORA-%d] No hay procesadoras activas. Actividad descartada", c.ID)
		return &pb.ProcesarActividadResponse{
			Exito:   false,
			Mensaje: "No hay procesadoras activas",
		}, nil
	}

	// Selecciona una procesadora mediante Round Robin
	destino := c.AsignarActividad(procesadoras)

	log.Printf("[PROCESADORA-%d] Actividad %s asignada a %s",
		c.ID, req.Actividad.ActividadId, destino.Nombre)

	// Envia la actividad a la procesadora seleccionada
	err = c.EnviarActividad(destino, req)
	if err != nil {
		log.Printf("[PROCESADORA-%d] Error enviando actividad %s a %s: %v",
			c.ID, req.Actividad.ActividadId, destino.Nombre, err)

		return &pb.ProcesarActividadResponse{
			Exito:   false,
			Mensaje: "Error enviando actividad a la procesadora asignada",
		}, err
	}

	return &pb.ProcesarActividadResponse{
		Exito:   true,
		Mensaje: fmt.Sprintf("Actividad asignada a la procesadora %s", destino.Nombre),
	}, nil
}

func (c *Core) ObtenerProcesadorasActivas() ([]*ns.Servicio, error) {
	// Obtiene las procesadoras registradas y activas en el NameServer.

	servicios, err := c.Naming.ListarProcesadoras()
	if err != nil {
		return nil, err
	}

	// Ordena las procesadoras por identificador.
	sort.Slice(servicios, func(i, j int) bool {
		var idI, idJ int

		fmt.Sscanf(servicios[i].Nombre, prefijoProcesadorasCore+"%d", &idI)
		fmt.Sscanf(servicios[j].Nombre, prefijoProcesadorasCore+"%d", &idJ)

		return idI < idJ
	})

	return servicios, nil
}

func (c *Core) AsignarActividad(
	procesadoras []*ns.Servicio,
) *ns.Servicio {
	// Selecciona la siguiente con RR
	c.mu.Lock()
	defer c.mu.Unlock()

	// Busca la siguiente
	for _, p := range procesadoras {
		var id int
		fmt.Sscanf(p.Nombre, prefijoProcesadorasCore+"%d", &id)

		if id > c.ultimoAsignado {
			c.ultimoAsignado = id
			return p
		}
	}

	// Si no hay una superior vuelve al inicio
	destino := procesadoras[0]

	var id int
	fmt.Sscanf(destino.Nombre, prefijoProcesadorasCore+"%d", &id)

	c.ultimoAsignado = id

	return destino
}

func (c *Core) EnviarActividad(
	destino *ns.Servicio,
	req *pb.ProcesarActividadRequest,
) error {
	// manda la actividad a la procesadora seleccionada
	var idDestino int
	_, err := fmt.Sscanf(
		destino.Nombre,
		prefijoProcesadorasCore+"%d",
		&idDestino,
	)
	if err != nil {
		return err
	}

	// Si la actividad fue asignada a si misma, la procesa directamente.
	if idDestino == c.ID {
		log.Printf("[PROCESADORA-%d] Procesando actividad %s localmente",
			c.ID, req.Actividad.ActividadId)

		respuesta, err := c.ProcesarActividad(context.Background(), req)
		if err != nil {
			return err
		}
		if !respuesta.Exito {
			log.Printf("[PROCESADORA-%d] Actividad %s no pudo entregarse: %s",
				c.ID, req.Actividad.ActividadId, respuesta.Mensaje)
		}
		return nil
	}

	// Resuelve la direccion de la procesadora mediante el NameServer.
	conn, err := grpc.NewClient(
		destino.Direccion,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	cliente := pb.NewProcesamientoClient(conn)

	// Establece un timeout para la entrega.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("[PROCESADORA-%d] Enviando actividad %s a %s",
		c.ID, req.Actividad.ActividadId, destino.Nombre)

	respuesta, err := cliente.ProcesarActividad(ctx, req)
	if err != nil {
		return err
	}

	if !respuesta.Exito {
		log.Printf("[PROCESADORA-%d] %s informó que la actividad %s no pudo entregarse: %s",
			c.ID, destino.Nombre, req.Actividad.ActividadId, respuesta.Mensaje)
	}

	return nil
}

// func (c *Core) ProcesarActividad(
// 	ctx context.Context,
// 	req *pb.ProcesarActividadRequest,
// ) (*pb.ProcesarActividadResponse, error) {
// 	// recibe una actividad asignada y la entrega a cuartel

// 	if req == nil || req.Actividad == nil {
// 		return &pb.ProcesarActividadResponse{
// 			Exito:   false,
// 			Mensaje: "La actividad recibida es invalida",
// 		}, nil
// 	}

// 	log.Printf("[PROCESADORA-%d] Actividad recibida para procesamiento: ID=%s",
// 		c.ID, req.Actividad.ActividadId)

// 	// direccion de cuartel
// 	direccionCuartel, err := c.Naming.ResolverCuartel()
// 	if err != nil {
// 		log.Printf("[PROCESADORA-%d] Cuartel no encontrado en NameServer. Actividad %s descartada: %v",
// 			c.ID, req.Actividad.ActividadId, err)

// 		return &pb.ProcesarActividadResponse{
// 			Exito:   false,
// 			Mensaje: "Cuartel no encontrado. Actividad descartada",
// 		}, nil
// 	}

// 	// conexion con cuartel
// 	conn, err := grpc.NewClient(
// 		direccionCuartel,
// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
// 	)
// 	if err != nil {
// 		log.Printf("[PROCESADORA-%d] Error conectando con Cuartel. Actividad %s descartada: %v",
// 			c.ID, req.Actividad.ActividadId, err)

// 		return &pb.ProcesarActividadResponse{
// 			Exito:   false,
// 			Mensaje: "Error conectando con Cuartel. Actividad descartada",
// 		}, nil
// 	}
// 	defer conn.Close()

// 	cliente := pb.NewProcesamientoClient(conn)

// 	// manda la actividad a cuartel
// 	ctxEntrega, cancel := context.WithTimeout(ctx, 5*time.Second)
// 	defer cancel()

// 	respuesta, err := cliente.ProcesarActividad(ctxEntrega, req)
// 	if err != nil {
// 		log.Printf("[PROCESADORA-%d] Cuartel no respondió. Actividad %s descartada: %v",
// 			c.ID, req.Actividad.ActividadId, err)

// 		return &pb.ProcesarActividadResponse{
// 			Exito:   false,
// 			Mensaje: "Cuartel no respondió. Actividad descartada",
// 		}, nil
// 	}

// 	log.Printf("[PROCESADORA-%d] Actividad %s entregada al Cuartel",
// 		c.ID, req.Actividad.ActividadId)

// 	return respuesta, nil
// }

func (c *Core) ProcesarActividad(
	ctx context.Context,
	req *pb.ProcesarActividadRequest,
) (*pb.ProcesarActividadResponse, error) {

	if req == nil || req.Actividad == nil {
		return &pb.ProcesarActividadResponse{
			Exito:   false,
			Mensaje: "La actividad recibida es invalida",
		}, nil
	}

	actividadID := req.Actividad.ActividadId

	log.Printf("[PROCESADORA-%d] Actividad recibida para procesamiento: ID=%s",
		c.ID, actividadID)

	// Reintenta hasta que cuartel confirme la recepción
	for {
		// Verifica si la solicitud original sigue vigente
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		// Resuelve nuevamente la dirección del Cuartel
		direccionCuartel, err := c.Naming.ResolverCuartel()
		if err != nil {
			log.Printf("[PROCESADORA-%d] No se pudo encontrar al Cuartel. Actividad %s. Reintentando...",
				c.ID, actividadID)
		} else {

			// Establece la conexión
			conn, err := grpc.NewClient(
				direccionCuartel,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)

			if err != nil {
				log.Printf("[PROCESADORA-%d] Error conectando con Cuartel: %v",
					c.ID, err)
			} else {
				cliente := cuartelpb.NewCuartelClient(conn)

				actividad := &cuartelpb.Actividad{
					ActividadId:  req.Actividad.ActividadId,
					Ubicacion:    req.Actividad.Ubicacion,
					Peligrosidad: req.Actividad.Peligrosidad,
					Integrantes:  req.Actividad.Integrantes,
					FechaEmision: req.Actividad.FechaEmision,
				}

				ctxEntrega, cancel := context.WithTimeout(ctx, 5*time.Second)

				_, err = cliente.RecibirActividad(
					ctxEntrega,
					&cuartelpb.RecibirActividadRequest{
						Actividad: actividad,
					},
				)

				cancel()
				conn.Close()

				if err == nil {
					log.Printf("[PROCESADORA-%d] Actividad %s entregada al Cuartel",
						c.ID, actividadID)

					return &pb.ProcesarActividadResponse{
						Exito:   true,
						Mensaje: "Actividad entregada al Cuartel",
					}, nil
				}

				log.Printf("[PROCESADORA-%d] Cuartel no respondió. Actividad %s. Reintentando: %v",
					c.ID, actividadID, err)
			}
		}

		// Espera antes de realizar un nuevo intento
		temporizador := time.NewTimer(3 * time.Second)

		select {
		case <-ctx.Done():
			temporizador.Stop()
			return nil, ctx.Err()

		case <-temporizador.C:
		}
	}
}
