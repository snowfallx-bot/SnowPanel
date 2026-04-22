import { http, unwrap } from "@/lib/http";
import {
  CreateDirectoryPayload,
  CreateDirectoryResult,
  DeleteFilePayload,
  DeleteFileResult,
  ListFilesResult,
  ReadTextFilePayload,
  ReadTextFileResult,
  WriteTextFilePayload,
  WriteTextFileResult
} from "@/types/file";

export function listFiles(path: string) {
  return unwrap<ListFilesResult>(http.get("/api/v1/files/list", { params: { path } }));
}

export function readTextFile(payload: ReadTextFilePayload) {
  return unwrap<ReadTextFileResult>(http.post("/api/v1/files/read", payload));
}

export function writeTextFile(payload: WriteTextFilePayload) {
  return unwrap<WriteTextFileResult>(http.post("/api/v1/files/write", payload));
}

export function createDirectory(payload: CreateDirectoryPayload) {
  return unwrap<CreateDirectoryResult>(http.post("/api/v1/files/mkdir", payload));
}

export function deleteFile(payload: DeleteFilePayload) {
  return unwrap<DeleteFileResult>(
    http.delete("/api/v1/files/delete", {
      data: payload
    })
  );
}
