Karina Aliaga 202373556-6
Nicolás Chehade 202373508-6

Los detalles de las VMs son:
VM1:
- Detalles:
  . Máquina utilizada: dist037
  . IP: 10.35.168.47
- Entidades dentro de la VM:
  1. Los Distros
  2. BBDD MySQL
- Instrucciones de ejecución:
  - up: sudo make docker-VM1
  - down: sudo make down-VM1


VM2:
- Detalles:
  . Máquina utilizada: dist038
  . IP: 10.35.168.48
- Entidades dentro de la VM:
  1. RabbitMQ
  2. Agente
  3. Procesadora1
- Instrucciones de ejecución:
  . up: sudo make docker-VM2
  . down: sudo make down-VM2

VM3:
- Detalles:
  . Máquina utilizada: dist039
  . IP: 10.35.168.49
- Entidades dentro de la VM:
  1. NameServer
  2. Procesadora2
- Instrucciones de ejecución:
  . up: sudo make docker-VM3
  . down: sudo make down-VM3


VM4:
- Detalles:
  . Máquina utilizada: dist040
  . IP: 10.35.168.50
- Entidades dentro de la VM:
  1. Cuartel
  2. Policía
  3. Procesadora3
- Instrucciones de ejecución:
  . up1: sudo make docker-VM4
  . up2: sudo make docker-VM4
  . down: sudo make down-VM4

Nota: 
VM4 cuenta con dos comandos de ejecución. Se pueden hacer en una misma terminal, pero no mostrará todos los logs. Se recomienda ejecutarlas en terminales por separado. Se hizo así para que policia tuviera una interfaz interactiva. 
