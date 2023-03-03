### Setup development

```shell
gem install dip
cp -n bidon_api/.env.sample bidon_api/.env || true
cp -n bidon_back/.env.sample bidon_back/.env || true

cd bidon_back
dip provision
dip rails c
dip rails s
dip bash

cd bidon_api
dip provision
dip rails c
dip rails s
dip bash
```
#### Clean env
```shell
docker compose down --volumes --rmi local --remove-orphans || true
```

### Start prod environment

#### Generate SECRET_KEY_BASE
```
docker compose -f docker-compose-prod.yml run --rm --no-deps bidon-backend rails secret
```
#### Deploy
Change the MaxMind credentials and SECRET_KEY_BASE.
```
docker compose -f docker-compose-prod.yml up -d
```
