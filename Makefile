docker-build-api:
	cd bidon_api && docker build -f Dockerfile_prod  -t registry.appodeal.com/bidon/api:$(TAG) .

docker-push-api:
	docker push registry.appodeal.com/bidon/api:$(TAG)

docker-build-back:
	cd bidon_back && docker build -f Dockerfile_prod -t registry.appodeal.com/bidon/back:$(TAG) .

docker-push-back:
	docker push registry.appodeal.com/bidon/back:$(TAG)
