import { http, unwrap } from "@/lib/http";
import {
  DockerContainerActionResult,
  ListDockerContainersResult,
  ListDockerImagesResult
} from "@/types/docker";
import { withEncodedSegment } from "@/api/path";

export function listDockerContainers() {
  return unwrap<ListDockerContainersResult>(http.get("/api/v1/docker/containers"));
}

export function startDockerContainer(id: string) {
  return unwrap<DockerContainerActionResult>(
    http.post(withEncodedSegment("/api/v1/docker/containers", id, "/start"))
  );
}

export function stopDockerContainer(id: string) {
  return unwrap<DockerContainerActionResult>(
    http.post(withEncodedSegment("/api/v1/docker/containers", id, "/stop"))
  );
}

export function restartDockerContainer(id: string) {
  return unwrap<DockerContainerActionResult>(
    http.post(withEncodedSegment("/api/v1/docker/containers", id, "/restart"))
  );
}

export function listDockerImages() {
  return unwrap<ListDockerImagesResult>(http.get("/api/v1/docker/images"));
}
