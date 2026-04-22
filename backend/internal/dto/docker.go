package dto

type DockerContainerActionPath struct {
	ID string `uri:"id" binding:"required"`
}

type DockerContainerInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Status string `json:"status"`
}

type ListDockerContainersResult struct {
	Containers []DockerContainerInfo `json:"containers"`
}

type DockerContainerActionResult struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

type DockerImageInfo struct {
	ID       string   `json:"id"`
	RepoTags []string `json:"repo_tags"`
	Size     uint64   `json:"size"`
}

type ListDockerImagesResult struct {
	Images []DockerImageInfo `json:"images"`
}
