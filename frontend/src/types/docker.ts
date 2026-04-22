export interface DockerContainerInfo {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
}

export interface ListDockerContainersResult {
  containers: DockerContainerInfo[];
}

export interface DockerContainerActionResult {
  id: string;
  state: string;
}

export interface DockerImageInfo {
  id: string;
  repo_tags: string[];
  size: number;
}

export interface ListDockerImagesResult {
  images: DockerImageInfo[];
}
