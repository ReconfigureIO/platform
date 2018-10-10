package main

import (
	"fmt"
	"sync"
)

// ContainerStatus represents the current status of a container.
type ContainerStatus int

const (
	StatusCreated ContainerStatus = iota
	StatusStarted
	StatusArchived
)

// DockerState exists to prevent a follower from attaching before a container
// has started, or after it has been deleted. It also prevents deletion while
// there exist log followers.
//
// This is needed because the docker logs command returns successfully with an
// empty log if a container has not yet started, and it's not particularly safe
// to delete a container whilst there are log followers.
type DockerState struct {
	mu sync.Mutex

	idToContainerState map[string]*ContainerState
}

func NewDockerState() *DockerState {
	return &DockerState{
		idToContainerState: map[string]*ContainerState{},
	}
}

// ContainerState represents the 'Created/Running/Archived' status of a
// container in a thread-safe manner. It keeps track of how many log followers
// hold a reference to the running docker container, and prevents deletion of
// that container until all of the followers have gone.
type ContainerState struct {
	mu sync.Mutex

	Cond           sync.Cond
	Status         ContainerStatus
	followersCount int
}

// Create inserts the given ID into the container state map, and returns the
// created *ContainerState.
func (ds *DockerState) Create(id string) *ContainerState {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	cs := &ContainerState{
		Status: StatusCreated,
	}
	cs.Cond.L = &cs.mu
	ds.idToContainerState[id] = cs
	return cs
}

// Delete marks a container as deleted. Its job is to avoid deleting the
// container in docker until the follower count reaches zero, and maintain the
// invariant that it remains zero once this function returns.
//
// As soon as this function is called, all future followers are told that this
// container is not known to the DockerState. They will therefore fall back to
// obtaining logs from the Archive. Therefore it must be arranged that logs
// exist in the archive before this function is called. However, there may be
// existing followers with valid references to the container in Docker until
// this function returns.
func (ds *DockerState) Delete(id string) {
	cs, success := ds.Get(id)
	if success != true {
		panic(fmt.Errorf("no such container during (*DockerState).Delete: %v", id))
	}

	// Future requests for this container will see present == false from Get().
	ds.mu.Lock()
	delete(ds.idToContainerState, id)
	ds.mu.Unlock()

	// Mark cs as archived
	cs.mu.Lock()
	cs.Status = StatusArchived
	cs.Cond.Broadcast() // Notify everyone that cs.Status changed.
	cs.mu.Unlock()

	// Block until all followers have gone.
	cs.mu.Lock()
	for cs.followersCount != 0 {
		cs.Cond.Wait()
	}
	cs.mu.Unlock()
}

// Get returns the container state for a given ID, returning present=false if
// the container was already deleted or never existed.
func (ds *DockerState) Get(id string) (cs *ContainerState, present bool) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	cs, present = ds.idToContainerState[id]
	return cs, present
}

// BecomeLogFollower increases the reference count to this container, preventing
// the container from being deleted until it is done with. However, it also
// blocks while the container is in the created state. It may return false to
// indicate that the container was deleted before we were able to obtain a valid
// reference. In that case the request should fall back to obtaining the logs
// from the archive.
func (cs *ContainerState) BecomeLogFollower() bool {
	// If the container has not yet started, block. This is required because
	// when you attach to a docker log, if the containers has not started yet,
	// it will complete immediately with success (rather than being an empty
	// log).
	cs.mu.Lock()
	for cs.Status == StatusCreated {
		cs.Cond.Wait()
	}
	if cs.Status == StatusArchived {
		// The container was archived by the time we woke up, so it's not
		// possible to attach. Downstream of here should fall back to the
		// archive.
		cs.mu.Unlock()
		return false
	}
	cs.followersCount++
	cs.mu.Unlock()
	return true
}

func (cs *ContainerState) UnfollowLogs() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.followersCount--
	cs.Cond.Broadcast()
}

func (cs *ContainerState) SetStarted() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.Status = StatusStarted
	cs.Cond.Broadcast()
}
