# Commands to build/start and stop the container.  Note this will not build locally.

composeup:
	docker-compose up --build

composedown:
	docker-compose down --volumes --rmi all
