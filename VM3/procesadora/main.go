package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

func main() {

	idTexto := os.Getenv("PROCESSOR_ID")
	puerto := os.Getenv("PROCESSOR_PORT")
	ip := os.Getenv("VM_IP")
	direccionNS := os.Getenv("NAMESERVER_ADDRESS")

	id, _ := strconv.Atoi(idTexto)

	// direccion ip:puerto
	direccionProcesadora := fmt.Sprintf("%s:%s", ip, puerto)

	// Crea el cliente gRPC del NameServer.
	naming, err := NewNamingClient(direccionNS, id, direccionProcesadora)
	if err != nil {
		log.Fatalf("[PROCESADORA-%d] Error creando cliente Naming: %v", id, err)
	}
	defer naming.Close()

	// Crea la instancia de Bully.
	bully := &Bully{
		ID:        id,
		Direccion: direccionProcesadora,
		Naming:    naming,
	}

	// Crea la instancia de Core.
	core := &Core{
		ID:     id,
		Bully:  bully,
		Naming: naming,
	}

	// Inicia el servidor
	servidor := &ServidorProcesadora{
		bully: bully,
		core:  core,
	}
	if err := servidor.IniciarServidor(":" + puerto); err != nil {
		log.Fatalf("[PROCESADORA-%d] Error iniciando servidor gRPC: %v", id, err)
	}

	// Registra la procesadora en el NameServer.
	for {
		if err := naming.RegistrarProcesadora(); err != nil {
			log.Printf("[PROCESADORA-%d] Error registrandose en NameServer: %v. Reintentando...", id, err)
			time.Sleep(3 * time.Second)
			continue
		}
		break
	}

	log.Printf("[PROCESADORA-%d] Registrada en NameServer como %s", id, direccionProcesadora)

	// Renueva el registro de la procesadora
	// Si aplica, también lo hacr con el del lider
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if err := naming.RenovarProcesadora(); err != nil {
				log.Printf("[PROCESADORA-%d] Error renovando registro: %v. Intentando registrar nuevamente...", id, err)
				if err := naming.RegistrarProcesadora(); err != nil {
					log.Printf("[PROCESADORA-%d] Error registrando nuevamente: %v", id, err)
				}
			}

			bully.mu.Lock()
			esLider := bully.LiderID == bully.ID
			bully.mu.Unlock()

			if esLider {
				if err := naming.RenovarLider(); err != nil {
					log.Printf("[PROCESADORA-%d] Error renovando alias de lider: %v. Intentando registrarlo nuevamente...", id, err)
					if err := naming.RegistrarLider(); err != nil {
						log.Printf("[PROCESADORA-%d] Error registrando alias de lider: %v", id, err)
					}
				}
			}
		}
	}()

	// Realiza la primera eleccion de lider.
	if err := bully.IniciarEleccion(); err != nil {
		log.Printf("[PROCESADORA-%d] Error en la eleccion inicial: %v", id, err)
	}

	// Inicia la vigilancia del lider.
	go bully.MonitorearLider()

	// Mantiene el proceso principal en ejecucion.
	select {}
}
