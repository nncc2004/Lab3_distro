package main

import (
	"context"
	"log"

	ns "lab3-shared/proto/nameserver"
	pb "lab3-shared/proto/procesadora"

	"fmt"

	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Bully struct {
	ID                 int
	Direccion          string
	Naming             *NamingClient
	LiderID            int
	DireccionLider     string
	ProcesadoraClient  pb.ProcesadoraClient
	mu                 sync.Mutex
	eleccionMu         sync.Mutex
	coordinadorVersion uint64
}

func (b *Bully) ActualizarLider(id int, direccion string) {
	// Actualiza la información del líder
	b.mu.Lock()
	defer b.mu.Unlock()

	b.LiderID = id
	b.DireccionLider = direccion
}

func (b *Bully) ObtenerProcesadorasActivas() ([]*ns.Servicio, error) {
	//Solo hace la consulta al NS
	return b.Naming.ListarProcesadoras()
}

func (b *Bully) EnviarEleccion(direccion string) (bool, error) {
	// Envía la solicitud ELECCION a una procesadora específica.
	conn, err := grpc.NewClient(
		direccion,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	cliente := pb.NewProcesadoraClient(conn)

	// Establece un timeout de 3 segundos para la respuesta.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	respuesta, err := cliente.Eleccion(
		ctx,
		&pb.EleccionRequest{
			IdProcesadora: int32(b.ID),
		},
	)
	if err != nil {
		return false, err
	}

	return respuesta.Aceptada, nil
}

func (b *Bully) AsumirLiderazgo() error {
	// se registra a si misma como lider en el NS y en el estado local
	err := b.Naming.RegistrarLider()
	if err != nil {
		return err
	}

	b.ActualizarLider(b.ID, b.Direccion)

	return nil
}

func (b *Bully) EnviarCoordinador() error {
	// Avisaa las demás procesadoras quién es el nuevo líder.
	procesadoras, err := b.ObtenerProcesadorasActivas()
	if err != nil {
		return err
	}

	for _, p := range procesadoras {
		if p.Direccion == b.Direccion {
			continue
		}

		conn, err := grpc.NewClient(
			p.Direccion,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			continue
		}

		cliente := pb.NewProcesadoraClient(conn)

		_, err = cliente.Coordinador(
			context.Background(),
			&pb.CoordinadorRequest{
				IdLider: int32(b.ID),
			},
		)

		conn.Close()

		if err != nil {
			continue
		}
	}

	return nil
}

func (b *Bully) RecibirCoordinador(ctx context.Context, req *pb.CoordinadorRequest,
) (*pb.CoordinadorResponse, error) {

	// recibe el anuncio y actualiza la información del líder conocido.

	procesadoras, err := b.ObtenerProcesadorasActivas()
	if err != nil {
		return nil, err
	}

	var direccionLider string

	for _, p := range procesadoras {
		// buscar la procesadora con ID del líder anunciado
		if p.Nombre == fmt.Sprintf("gamma.procesamiento.procesador%d", req.IdLider) {
			direccionLider = p.Direccion
			break
		}
	}

	if direccionLider == "" {
		return nil, fmt.Errorf("no se encontró la dirección del líder %d", req.IdLider)
	}

	// Actualiza
	b.ActualizarLider(int(req.IdLider), direccionLider)

	b.mu.Lock()
	b.coordinadorVersion++
	b.mu.Unlock()

	return &pb.CoordinadorResponse{
		Recibido: true,
	}, nil
}

func (b *Bully) IniciarEleccion() error {

	// Evita elecciones simultáneas.
	b.eleccionMu.Lock()
	defer b.eleccionMu.Unlock()

	for {
		// Guarda la versión actual de las notificaciones coordinador.
		b.mu.Lock()
		versionInicial := b.coordinadorVersion
		b.mu.Unlock()

		procesadoras, err := b.ObtenerProcesadorasActivas()
		if err != nil {
			return err
		}

		hayProcesadoraSuperior := false
		recibioOK := false

		// Busca procesadoras con ID superior y les envía eleccion.
		for _, p := range procesadoras {
			var id int

			_, err := fmt.Sscanf(
				p.Nombre,
				"gamma.procesamiento.procesador%d",
				&id,
			)

			if err != nil || id <= b.ID {
				continue
			}

			hayProcesadoraSuperior = true

			aceptada, err := b.EnviarEleccion(p.Direccion)

			if err == nil && aceptada {
				recibioOK = true
			}
		}

		// como ninguna procesadora superior responde OK, se hace lider
		if !hayProcesadoraSuperior || !recibioOK {
			if err := b.AsumirLiderazgo(); err != nil {
				return err
			}

			return b.EnviarCoordinador()
		}

		log.Printf("Se recibió OK. Esperando COORDINADOR...")

		temporizador := time.NewTimer(3 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)

		esperando := true

		for esperando {
			select {
			case <-ticker.C:
				b.mu.Lock()
				recibido := b.coordinadorVersion > versionInicial
				b.mu.Unlock()

				if recibido {
					esperando = false
				}

			case <-temporizador.C:
				log.Printf("No llegó COORDINADOR. Reiniciando elección...")
				esperando = false
			}
		}

		ticker.Stop()
		temporizador.Stop()

		b.mu.Lock()
		recibido := b.coordinadorVersion > versionInicial
		b.mu.Unlock()

		//Si llegó la notificación, retorna
		if recibido {
			return nil
		}

	}
}

func (b *Bully) RecibirEleccion(ctx context.Context, req *pb.EleccionRequest,
) (*pb.EleccionResponse, error) {
	// responde OK a una procesadora de menor ID e inicia una elección propia

	if int(req.IdProcesadora) >= b.ID {
		return &pb.EleccionResponse{
			Aceptada: false,
		}, nil
	}

	// aquí inicia la eleccion
	go func() {
		if err := b.IniciarEleccion(); err != nil {
			log.Printf("Error al iniciar elección: %v", err)
		}
	}()

	return &pb.EleccionResponse{
		Aceptada: true,
	}, nil
}

func (b *Bully) MonitorearLider() {
	//Hace el ping cada 3 segundos al lider que tiene almacenado.
	//Si no tiene ninguno (al inicio por ejempo) inmediatamente comienza la votación

	for {
		b.mu.Lock()
		liderID := b.LiderID
		direccionLider := b.DireccionLider
		b.mu.Unlock()

		if liderID == 0 || direccionLider == "" {
			log.Printf("No hay un líder conocido. Iniciando elección...")
			if err := b.IniciarEleccion(); err != nil {
				log.Printf("Error al iniciar elección: %v", err)
			}
			continue
		}

		time.Sleep(3 * time.Second)

		// Actualiza la información por si cambió el líder durante la espera.
		b.mu.Lock()
		liderID = b.LiderID
		direccionLider = b.DireccionLider
		b.mu.Unlock()

		if liderID == b.ID {
			continue
		}

		conn, err := grpc.NewClient(
			direccionLider,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)

		if err != nil {
			log.Printf("Error conectando con el líder: %v", err)
			continue
		}

		cliente := pb.NewProcesadoraClient(conn)

		_, err = cliente.Ping(
			context.Background(),
			&pb.PingRequest{},
		)

		conn.Close()

		// implementé que revise dos vecees por si acaso falla algo
		// No me compliqué, son solo dos llamadas. Si fallan las dos seguidas
		// Se asume que no hay líder
		if err != nil {
			log.Printf("El líder no respondió al primer Ping. Reintentando...")

			conn, err = grpc.NewClient(
				direccionLider,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)

			if err != nil {
				b.IniciarEleccion()
				continue
			}

			cliente = pb.NewProcesadoraClient(conn)

			_, err = cliente.Ping(
				context.Background(),
				&pb.PingRequest{},
			)

			conn.Close()

			// Aquí lo del segund fallo
			if err != nil {
				log.Printf("El líder no respondió al segundo Ping. Iniciando elección...")
				if err := b.IniciarEleccion(); err != nil {
					log.Printf("Error al iniciar elección: %v", err)
				}
			}
		}
	}
}
