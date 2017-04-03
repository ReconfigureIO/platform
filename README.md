# platform
The backend of Reconfigure.io

## Developing

1. Install [Docker Compose](https://docs.docker.com/compose/overview/)
2. run `docker-compose up` in the top level directory.
3. `curl http://localhost:8080/ping`

# API

## Schema

All API access is over HTTP and is (after running `docker-compse up`) accessible on `localhost:8080`. Data is sent and received as JSON.

Blank fields are included as `null` rather than being ommitted

Timestamps are returned in ISO 8601 format:
```
YYYY-MM-DDTHH:MM:SSZ
```

## Resources

### Projects
`Projects` are collections of builds all with a common theme and owned by one user, you can list Projects like so:

#### GET /projects

```
curl -X GET localhost:8080/projects
{"projects":[{"id":1,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":2,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":3,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":4,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":5,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null},{"id":6,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null}]}
```

#### POST /projects

Create a new project

Projects have a UserID and a Name

```
curl -X POST -F 'name=project-name' -F 'user_id=1' http://localhost:8080/projects
```

You can expect this to return a HTTP `201` code with the newly created project including ID

<TODO> Describe format, return codes (201)

#### GET /projects/{project_id}

To view one project's details, specify the `ID` of that project:
```
curl -X GET localhost:8080/projects/3
{"id":3,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"name":"parallel-histogram","builds":null}
```

#### PUT /projects/{project_id}

Change the name of a project, assign project to another user(? useful for organisations but not right now)

```
curl -X PUT -F 'name=project-name' -F 'user_id=1' http://localhost:8080/projects/{project_id}
```
<TODO> Describe format, return codes (204)

### Builds
`Builds` are one run of user files through our compiler. They have input artifacts and output streams along with a status, they may also have output artifacts. To list all builds:

#### GET /builds

```
curl -X GET localhost:8080/builds
{"builds":[{"id":1,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"project":{"id":0,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":0,"name":"","builds":null},"project_id":0,"input_artifact":"golang code","output_artifact":".bin file","outout_stream":"working working done","status":""}]}

```

#### GET /builds/{build_id}

To view one particular build's details:

```
curl -X GET localhost:8080/builds/1
{"id":1,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"project":{"id":0,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":0,"name":"","builds":null},"project_id":0,"input_artifact":"golang code","output_artifact":".bin file","outout_stream":"working working done","status":""}
```

#### GET /builds?project={project_id}

To view all of the builds associated with a project do the following:
```
curl -X GET localhost:8080/builds?project=0
{"builds":[{"id":1,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":1,"project":{"id":0,"user":{"id":0,"github_id":"","email":"","auth_token":null},"user_id":0,"name":"","builds":null},"project_id":0,"input_artifact":"golang code","output_artifact":".bin file","outout_stream":"working working done","status":""}]}
```

#### POST /builds

Builds have a UserID, ProjectID, InputArtifact, OutputArtifact, OutputStream and a Status. OutputArtifact and OutputStream are optional.

```
curl -X POST -F 'user_id=1' -F 'project_id=1' -F 'input_artifact=s3://somefile.tar.gz' -F 'status=SUBMITTED' http://localhost:8080/builds
```

You can expect this to return a HTTP `202` code with the newly created build including ID

#### PUT /builds/{id}

You can update a Build for instance when its status changes or the output artifact needs setting.

```
curl -X PUT -F 'user_id=1' -F 'project_id=1' -F 'input_artifact=s3://somefile.tar.gz' -F 'output_artifact=s3://somefile.tar.gz' -F 'status=COMPLETE' http://localhost:8080/builds/1
```
You can expect this to return a HTTP `204` code

#### GET /builds/{build_id}/logs

Stream the logs for a given build

<TODO> Describe format, termination

#### GET /builds/{build_id}/logs

Stream the logs for a given build

<TODO> Describe format, termination

## What to expect
In the event of an invalid ID we can expect to receive a `404` response from the API:

```
curl -vvvv -X GET localhost:8080/users/foo
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
