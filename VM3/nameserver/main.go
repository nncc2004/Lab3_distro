/*
Esto no deja de ser un diccionario en memoria con 4 operaciones:
1. Registrar
2. Renovar
3. Resolver
4. Listar

Además, deberá peroidicamente revisar cuáles direcciones
expiraron (10s).
Notar que son las entidades las encargadas de registrarse cada 3s, aquí
sólo debo implmentar la función para que lo hagan.

Habrá que usar un candado para el acceso concurrente por entidades
al diccionario.
*/

package main

import (
	"context"
	"errors"
	pb "lab3-shared/proto/nameserver"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Structs
type registro struct {
	direccion  string
	expiracion time.Time
}

type diccionario struct {
	mu        sync.Mutex
	registros map[string]registro
}

//Métodos 'públicos' (uno por operación)

func (d *diccionario) registrar(nombre string, direccion string) error {
	/*
		Recibe el nombre y dirección.
		Si el nombre ya existe, retorna error de que ya existe (no actualiza)
		Si el nombre no existe, lo registra.
	*/
	if nombre == "" || direccion == "" {
		return errors.New("nombre o direccion vacios")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.registros[nombre] = registro{
		direccion:  direccion,
		expiracion: time.Now().Add(10 * time.Second),
	}

	log.Printf("[NAMESERVER] Registro creado o actualizado: %s -> %s", nombre, direccion)
	return nil

	// if _, existe := d.registros[nombre]; existe {
	// 	return errors.New("el nombre ya se encuentra registrado")
	// }

	// d.registros[nombre] = registro{
	// 	direccion:  direccion,
	// 	expiracion: time.Now().Add(10 * time.Second),
	// }
	// log.Printf("[NAMESERVER] Registro creado: %s -> %s", nombre, direccion)
	// return nil
}

func (d *diccionario) renovar(nombre string) error {
	/*
		Sólo renueva. SI no existe el registro, retorna error.
	*/
	if nombre == "" {
		return errors.New("nombre vacio")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	registro, existe := d.registros[nombre]

	if !existe {
		return errors.New("nombre no registrado")
	}

	if !time.Now().Before(registro.expiracion) {
		return errors.New("registro expirado")
	}

	registro.expiracion = time.Now().Add(10 * time.Second)
	d.registros[nombre] = registro
	log.Printf("[NAMESERVER] Registro renovado: %s", nombre)
	return nil
}

func (d *diccionario) resolver(nombre string) (string, error) {
	/*
		Recibe el nombre (como gamma.procesamiento.procesador1) y verifica si está.
		Si existe, verifica si está expirado o no.
		Si pasa rodo, retorna la dirección. Sino, retorna un error según corresponda.
	*/
	if nombre == "" {
		return "", errors.New("nombre vacio")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	registro, existe := d.registros[nombre]

	if !existe {
		return "", errors.New("nombre no registrado")
	}

	if !time.Now().Before(registro.expiracion) {
		return "", errors.New("registro expirado")
	}
	log.Printf("[NAMESERVER] Nombre resuelto: %s -> %s", nombre, registro.direccion)
	return registro.direccion, nil
}

func (d *diccionario) listar(prefijo string) (map[string]string, error) {
	/*
		Recibe un prefijo (como gamma.procesamiento) y retorna todas los registros
		cuyos nombres coincidan con ese prefijo. Nuevamente, verifica si está expirado.
		Notar que aquí las procesadoras se podrían dar cuenta de que no hay un líder,
		al hacer el llamado y notar que no está.
	*/
	if prefijo == "" {
		return nil, errors.New("prefijo vacio")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	resultado := make(map[string]string)

	for nombre, registro := range d.registros {

		if !strings.HasPrefix(nombre, prefijo) {
			continue
		}

		if !time.Now().Before(registro.expiracion) {
			continue
		}

		resultado[nombre] = registro.direccion
	}
	log.Printf("[NAMESERVER] Listado para prefijo '%s': %d servicios encontrados", prefijo, len(resultado))
	return resultado, nil
}

// Método 'privado'
func (d *diccionario) expirarRegistros() {
	/*
		Revisa uno a uno todos los registros, si hay uno expirado, lo borra completo.
	*/
	d.mu.Lock()
	defer d.mu.Unlock()

	ahora := time.Now()

	for nombre, registro := range d.registros {

		if !ahora.Before(registro.expiracion) {
			log.Printf("[NAMESERVER] Registro expirado y eliminado: %s", nombre)
			delete(d.registros, nombre)
		}
	}
}

// Lógica del gRPC
type nameserverServer struct {
	pb.UnimplementedNameServerServer
	diccionario *diccionario
}

// Métodos para el gRPC
func (s *nameserverServer) RegistrarRPC(ctx context.Context,
	req *pb.RegistrarRequest,
) (*pb.RegistrarResponse, error) {

	err := s.diccionario.registrar(req.Nombre, req.Direccion)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.RegistrarResponse{}, nil
}

func (s *nameserverServer) RenovarRPC(
	ctx context.Context,
	req *pb.RenovarRequest,
) (*pb.RenovarResponse, error) {

	err := s.diccionario.renovar(req.Nombre)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.RenovarResponse{}, nil
}

func (s *nameserverServer) ResolverRPC(
	ctx context.Context,
	req *pb.ResolverRequest,
) (*pb.ResolverResponse, error) {

	direccion, err := s.diccionario.resolver(req.Nombre)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &pb.ResolverResponse{
		Direccion: direccion,
	}, nil
}

func (s *nameserverServer) ListarRPC(
	ctx context.Context,
	req *pb.ListarRequest,
) (*pb.ListarResponse, error) {

	registros, err := s.diccionario.listar(req.Prefijo)

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	respuesta := &pb.ListarResponse{}

	for nombre, direccion := range registros {
		respuesta.Servicios = append(
			respuesta.Servicios,
			&pb.Servicio{
				Nombre:    nombre,
				Direccion: direccion,
			},
		)
	}

	return respuesta, nil
}

func main() {

	puerto := os.Getenv("NAMESERVER_PORT")
	if puerto == "" {
		puerto = "50051"
	}

	// dicc
	d := &diccionario{
		registros: make(map[string]registro),
	}

	// Crear servidor gRPC
	servidor := grpc.NewServer()

	pb.RegisterNameServerServer(servidor, &nameserverServer{
		diccionario: d,
	})

	// Abrir puerto
	listener, err := net.Listen("tcp", ":"+puerto)
	if err != nil {
		log.Fatalf("[NAMESERVER] Error al abrir puerto: %v", err)
	}

	// expiración
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			d.expirarRegistros()
		}
	}()

	log.Printf("[NAMESERVER] Escuchando en puerto %s", puerto)

	// Iniciar servidor
	if err := servidor.Serve(listener); err != nil {
		log.Fatalf("[NAMESERVER] Error en servidor gRPC: %v", err)
	}
}
