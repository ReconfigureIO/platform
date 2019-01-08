# platform
The backend of Reconfigure.io

## Developing

1. Install [Docker Compose](https://docs.docker.com/compose/overview/)
2. run `docker-compose up` in the top level directory.
3. `curl http://localhost:8080/ping`

## Signing Up (without a Token)

1. Visit `http://localhost:8080/oauth/new-account`
2. Login with Github
3. Use the generated token with our tooling
4. If you need to view this token again visit https://localhost:8080/oauth/signin
5. Optional `redirect_url` query param to get redirected to specific url after login.

## Logging out

Visit `https://api.reconfigure.io/oauth/logout`. This will return a 204.

## Developing On-Premises

1. Install [Docker Compose](https://docs.docker.com/compose/overview/)
2. run `docker-compose -f docker-compose.on-prem.yml up` in the top level directory.
3. `curl http://localhost:8080/ping`

## Signing Up for On-Premises

1. Visit `http://localhost:8080/`
2. Enter an email address
3. Use the generated token with our tooling
4. If you need to view this token again visit https://localhost:8080/
5. Optional `redirect_url` query param to get redirected to specific url after login.
