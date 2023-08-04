REGISTRY = ghcr.io/bidon-io

docker-build-push-prod-back:
	cd bidon_back && \
	docker buildx build --platform linux/amd64  \
  --target prod --cache-to $(REGISTRY)/bidon-back --cache-from $(REGISTRY)/bidon-back \
	-t $(REGISTRY)/bidon-back:$(TAG) -t $(REGISTRY)/bidon-back:latest .

docker-build-push-prod-admin:
	docker buildx build --platform linux/amd64 \
	--target bidon-admin --cache-to $(REGISTRY)/bidon-admin --cache-from $(REGISTRY)/bidon-admin \
	-t $(REGISTRY)/bidon-admin:$(TAG) -t $(REGISTRY)/bidon-admin:latest .

docker-build-push-prod-sdkapi:
	docker buildx build --platform linux/amd64 \
	--target bidon-sdkapi --cache-to $(REGISTRY)/bidon-sdkapi --cache-from $(REGISTRY)/bidon-sdkapi \
	-t $(REGISTRY)/bidon-sdkapi:$(TAG) -t $(REGISTRY)/bidon-sdkapi:latest .
