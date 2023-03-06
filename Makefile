docker-build-prod-api:
	#cd bidon_api && docker build --target=prod -t registry.appodeal.com/bidon/api:$(TAG) -t registry.appodeal.com/bidon/api:latest -t ghcr.io/bidon-io/bidon-api:latest -t ghcr.io/bidon-io/bidon-api:$(TAG) .
	cd bidon_api && docker buildx build --platform linux/amd64,linux/arm64 --target=prod --cache-from ghcr.io/bidon-io/bidon-api:latest -t registry.appodeal.com/bidon/api:$(TAG) -t registry.appodeal.com/bidon/api:latest -t ghcr.io/bidon-io/bidon-api:latest -t ghcr.io/bidon-io/bidon-api:$(TAG) --push .
docker-push-prod-api:
	docker push registry.appodeal.com/bidon/api:$(TAG)
	docker push registry.appodeal.com/bidon/api:latest
	docker push ghcr.io/bidon-io/bidon-api:$(TAG)
	docker push ghcr.io/bidon-io/bidon-api:latest

docker-build-prod-back:
	#cd bidon_back && docker build --target=prod -t registry.appodeal.com/bidon/back:$(TAG) -t registry.appodeal.com/bidon/back:latest -t ghcr.io/bidon-io/bidon-back:latest -t ghcr.io/bidon-io/bidon-back:$(TAG) .
	cd bidon_api && docker buildx build --platform linux/amd64,linux/arm64 --target=prod -t registry.appodeal.com/bidon/back:$(TAG) -t registry.appodeal.com/bidon/back:latest -t ghcr.io/bidon-io/bidon-back:latest -t ghcr.io/bidon-io/bidon-back:$(TAG) --push .

docker-push-prod-back:
	docker push registry.appodeal.com/bidon/back:$(TAG)
	docker push registry.appodeal.com/bidon/back:latest
	docker push ghcr.io/bidon-io/bidon-back:$(TAG)
	docker push ghcr.io/bidon-io/bidon-back:latest
