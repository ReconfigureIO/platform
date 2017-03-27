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

```
curl localhost:8080/projects
{"projects":[{"ID":1,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":2,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":3,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":4,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":5,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null},{"ID":6,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null}]}
```

To view one project's details, specify the `ID` of that project:
```
curl localhost:8080/projects/3
{"Project":{"ID":3,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":1,"Name":"parallel-histogram","Builds":null}}
```

To view all of the builds associated with a project do the following:
```
curl localhost:8080/projects/1/builds
{"Builds":{"ID":0,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":0,"Project":{"ID":0,"User":{"ID":0,"GithubID":"","Email":"","AuthTokens":null},"UserID":0,"Name":"","Builds":null},"ProjectID":0,"InputArtifact":"","OutputArtifact":"","OutputStream":"","Status":""}}
```

### Builds
list, details, status

### Users
list, details, projects

## What to expect
404 if an ID's invalid