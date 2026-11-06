.PHONY: docker-VM4 policia

docker-VM4:
	docker compose down -v
	docker compose build --no-cache
	docker compose up -d
	docker compose logs -f cuartel procesadora
	
policia:
	docker compose run --rm policia

down-VM4:
	docker compose down