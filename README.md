# platform
The backend of Reconfigure.io

## Developing

1. Install [Docker Compose](https://docs.docker.com/compose/overview/)
2. run `docker-compose up` in the top level directory.
3. `curl http://localhost:8080/ping`

## Creating an invite token

```
curl -v -XPOST https://admin:ffea108b2166081bcfd03a99c597be78b3cf30de685973d44d3b86480d644264@api.reconfigure.io/admin/invites
{"value":{"token":"Jroy3dkYHATiDtU3cPpJQWYEyvkGSasIXnXPHgMxI62PliENinA4lUAwwi051UZl","created_at":"2017-04-17T17:34:04.768483806Z"}}
```

## Signing Up

1. Visit https://api.reconfigure.io/oauth/signin/<invite token>
2. Login with Github

# API

## Schema

All API access is over HTTP and is (after running `docker-compse up`) accessible on `localhost:8080`. Data is sent and received as JSON.

Blank fields are included as `null` rather than being ommitted

Timestamps are returned in ISO 8601 format:
```
YYYY-MM-DDTHH:MM:SSZ
```

## Responses
Success
All success responses have a `value` field holding the response.
```
{"value": {...}}
```
All error responses have `error` field holding the error message.
```
{"error": "some error message"}
```

## Resources

### Projects
`Projects` are collections of builds all with a common theme and owned by one user, you can list Projects like so:

#### GET /projects

```
curl -u $USER:$PASS -X GET localhost:8080/projects
{"value":[{"id":1,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":2,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":3,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":4,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":5,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":6,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null}]}
```

#### POST /projects

Create a new project

Projects have a UserID and a Name

```
curl -u $USER:$PASS -H "Content-Type: application/json" -X POST -d '{"name":"addition", "user_id":1}' http://localhost:8080/projects
```

You can expect this to return a HTTP `201` code with the newly created project including ID

<TODO> Describe format, return codes (201)

#### GET /projects/{project_id}

To view one project's details, specify the `ID` of that project:
```
curl -u $USER:$PASS -H GET localhost:8080/projects/3
{"value":{"id":3,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null}}
```

#### PUT /projects/{project_id}

Change the name of a project, assign project to another user(? useful for organisations but not right now)

```
curl -u $USER:$PASS -H "Content-Type: application/json" -X PUT -d '{"name":"addition", "user_id":1}' http://localhost:8080/projects/1
```
<TODO> Describe format, return codes (204)

### Builds
`Builds` are one run of user files through our compiler. They have input artifacts and output streams along with a status, they may also have output artifacts. To list all builds:

#### GET /builds

```
curl -u $USER:$PASS -H GET localhost:8080/builds
{"value":[{"id":56,"project":{"id":87,"name":"addition"},"job":{"events":[{"timestamp":"2017-04-12T21:41:13.273744Z","status":"QUEUED","code":0},{"timestamp":"2017-04-12T21:41:18.358054Z","status":"STARTED","code":0},{"timestamp":"2017-04-12T21:41:28.356844Z","status":"COMPLETED","code":0}]}}]}

```

#### GET /builds/{build_id}

To view one particular build's details:

```
curl -u $USER:$PASS -H GET localhost:8080/builds/1
{"value": {"id":56,"project":{"id":87,"name":"addition"},"job":{"events":[{"timestamp":"2017-04-12T21:41:13.273744Z","status":"QUEUED","code":0},{"timestamp":"2017-04-12T21:41:18.358054Z","status":"STARTED","code":0},{"timestamp":"2017-04-12T21:41:28.356844Z","status":"COMPLETED","code":0}]}}}
```

#### GET /builds?project={project_id}

To view all of the builds associated with a project do the following:
```
curl -u $USER:$PASS -H GET localhost:8080/builds?project=0
{"value":[{"id":56,"project":{"id":1,"name":"addition"},"job":{"events":[{"timestamp":"2017-04-12T21:41:13.273744Z","status":"QUEUED","code":0},{"timestamp":"2017-04-12T21:41:18.358054Z","status":"STARTED","code":0},{"timestamp":"2017-04-12T21:41:28.356844Z","status":"COMPLETED","code":0}]}}]}
```

#### POST /builds

Creates a new build in an `SUBMITTED` status.

```
curl -u $USER:$PASS -H "Content-Type: application/json" -X POST -d '{"user_id":1, "project_id":1}' http://localhost:8080/builds
```

You can expect this to return a HTTP `202` code with the newly created build including ID

#### PUT  {{ build.input_url }}

Attached the enclosed input to a build, moving it to `QUEUED` status

```
curl -v -XPUT --data-binary @../examples/addition/.reco-work/bundle.tar.gz http://localhost:8080/builds/1/input
```

You can expect this to return  a HTTP `204` code.

#### POST /builds/1/events

Allows creation of events, moving from one state to another.

For users, the most relevent is `TERMINATED`, which will stop any running jobs.

```
curl -v -XPOST -H "Content-Type: application/json"  -d '{"status": "TERMINATED"}' http://localhost:8080/builds/1/events
```

### Simulations
`Simulations` are one run of user files through our compiler. They have input artifacts and output streams along with a status, they never have output artifacts. To list all simulations:

#### GET /simulations
Gets a list of all simulations, can be filtered by project ID.

```
curl -X GET localhost:8080/simulations
{"value":[{"id":34,"project":{"id":93,"name":"addition"},"job":{"events":[{"timestamp":"2017-04-12T21:46:16.789615Z","status":"QUEUED","code":0},{"timestamp":"2017-04-12T21:46:21.872226Z","status":"STARTED","code":0},{"timestamp":"2017-04-12T21:46:31.872251Z","status":"COMPLETED","code":0}]},"command":"test-addition"}]}
```

#### POST /simulations

Creates a new simulation in an `SUBMITTED` status

Simulations have a ProjectID, a Command.

```
curl -X POST -H "Content-Type: application/json"  -d '{"project_id": 1, "cmd": "test-addition"}' http://localhost:8080/simulations
{"value":[{"id":1,"project":{"id":1,"name":"addition"},"command":"test-addition"}]}
```

You can expect this to return a HTTP `202` code with the newly created build including ID

#### PUT  {{ simulations.input_url }}

Attached the enclosed input to a simulation, moving it to `QUEUED` status

```
curl -v -XPUT --data-binary @../examples/addition/.reco-work/bundle.tar.gz http://localhost:8080/simulations/1/input
```

You can expect this to return  a HTTP `204` code.


#### GET /simulations/{id}

To view one particular simulation's details:

```
curl -X GET localhost:8080/simulation/1
{"value":{"id":34,"project":{"id":93,"name":"addition"},"job":{"events":[{"timestamp":"2017-04-12T21:46:16.789615Z","status":"QUEUED","code":0},{"timestamp":"2017-04-12T21:46:21.872226Z","status":"STARTED","code":0},{"timestamp":"2017-04-12T21:46:31.872251Z","status":"COMPLETED","code":0}]},"command":"test-addition"}}
```

#### GET /simulations/{id}/logs

Stream the logs for a given simulation

<TODO> Describe returned values


#### POST /simulations/1/events

Allows creation of events, moving from one state to another.

For users, the most relevent is `TERMINATED`, which will stop any running jobs.

```
curl -v -XPOST -H "Content-Type: application/json"  -d '{"status": "TERMINATED"}' http://localhost:8080/simulations/1/events
```

#### GET /builds/{build_id}/logs

Stream the logs for a given build

<TODO> Describe format, termination

#### GET /deployments

Get a list of deployments, can be filtered by parent `build ID`.

<TODO> examples
		
#### POST /deployments
		
Create and process a deployment. Requires a parent `build ID` and a `command`

<TODO> examples 
		
#### GET /deployments/{id}

Get the details of an individual deployment.

<TODO> examples
		
#### GET /deployments/{id}/logs
Stream the logs of an individual deployment.

<TODO> examples

## What to expect
In the event of an invalid ID we can expect to receive a `404` response from the API:

```
curl -v -X GET localhost:8080/builds/foo
Note: Unnecessary use of -X or --request, GET is already inferred.
*   Trying 127.0.0.1...
* Connected to localhost (127.0.0.1) port 8080 (#0)
> GET /users/foo HTTP/1.1
> Host: localhost:8080
> User-Agent: curl/7.47.0
> Accept: */*
>
< HTTP/1.1 404 Not Found
< Content-Length: 0
< Content-Type: text/plain; charset=utf-8
< Date: Mon, 27 Mar 2017 15:52:53 GMT
<
* Connection #0 to host localhost left intact
```
