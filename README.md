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
On `Mac M1` change `LD_PRELOAD: /usr/lib/aarch64-linux-gnu/libjemalloc.so` in `docker-compose-prod.yml`

Use the following command to generate `SECRET_KEY_BASE`:
```shell
docker compose -f docker-compose-prod.yml run --rm --no-deps bidon-backend rails secret
```
Create personal account on https://maxmind.com.

Change `GEOIPUPDATE_ACCOUNT_ID`, `GEOIPUPDATE_LICENSE_KEY`, `SECRET_KEY_BASE` or set on command and start Docker Compose:
```shell
MAXMIND_ACCOUNT_ID=********** \
MAXMIND_LICENSE_KEY=********** \
SECRET_KEY_BASE=4232a1c84d26f08101987a910b38387e64be09dd2a883634044c994c4d7e456b31286a61b1615606ba0ef71a80129b3e1fc1ba0d87d691a9f01c4d2f3bf5431b \
PG_PASSWORD=passwsord123 \
docker compose -f docker-compose-prod.yml up -d
```
