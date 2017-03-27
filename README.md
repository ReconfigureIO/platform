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
{"projects":[{"ID":1,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":2,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":3,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":4,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":5,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":6,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null}]}
```

#### POST /projects

Create a new project

<TODO> Describe format, return codes (201)

#### GET /projects/{project_id}

To view one project's details, specify the `ID` of that project:
```
curl -X GET localhost:8080/projects/3
{"Project":{"ID":3,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null}}
```

#### PUT /projects/{project_id}

Update a project

<TODO> Describe format, return codes (204)

#### GET /projects/{project_id}/builds

To view all of the builds associated with a project do the following:
```
curl -X GET localhost:8080/projects/1/builds
{"Builds":{"ID":0,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":0,"Project":{"ID":0,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":0,"Name":"","Builds":null},"ProjectID":0,"InputArtifact":"","OutputArtifact":"","OutputStream":"","Status":""}}
```

#### POST /projects/{project_id}/builds

Create a build for a project

<TODO> Describe format, return codes (201)

### Builds
`Builds` are one run of user files through our compiler. They have input artifacts and output streams along with a status, they may also have output artifacts. To list all builds:

#### GET /builds

```
curl -X GET localhost:8080/builds
{"Builds":{"ID":0,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":0,"Project":{"ID":0,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":0,"Name":"","Builds":null},"ProjectID":0,"InputArtifact":"","OutputArtifact":"","OutputStream":"","Status":""}}
```

#### GET /builds/{build_id}

To view one particular build's details:

```
curl -X GET localhost:8080/builds/1
{"build":{"ID":1,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Project":{"ID":0,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":0,"Name":"","Builds":null},"ProjectID":0,"InputArtifact":"golang code","OutputArtifact":".bin file","OutputStream":"working working done","Status":""}}
```

#### GET /builds/{build_id}/status

Sometimes we only need the status of a build, e.g. for reporting or for internal workflows:

```
curl -X GET localhost:8080/builds/30/status
{"status":"COMPLETED"}
```

#### GET /builds/{build_id}/logs

Stream the logs for a given build

<TODO> Describe format, termination


### Users

A `User` has an email address, Github username and many projects. We can list users by doing:

#### GET /users

```
curl -X GET localhost:8080/users
{"users":[{"ID":1,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":2,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":3,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":4,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":5,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":6,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":7,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":8,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":9,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":10,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":11,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":12,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":13,"GithubID":"foobar","Email":"","AuthTokens":null},{"ID":14,"GithubID":"helloworld","Email":"","AuthTokens":null},{"ID":15,"GithubID":"john","Email":"","AuthTokens":null},{"ID":16,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":17,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":18,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":19,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":20,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":21,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":22,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":23,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":24,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":25,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":26,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":27,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":28,"GithubID":"campgareth","Email":"","AuthTokens":null},{"ID":29,"GithubID":"campgareth","Email":"","AuthTokens":null}]}

```

#### GET /users/{user_id}

We can also view the details of one user like so:
```
curl -X GET localhost:8080/users/29
{"user":{"ID":29,"GithubID":"campgareth","Email":"","AuthTokens":null}}
```

#### GET /users/{user_id}/projects

We can view the projects associated with a user like so:
```
curl -X GET localhost:8080/users/1/projects
{"projects":[{"ID":1,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":2,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":3,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":4,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":5,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":6,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null}]}
```

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
