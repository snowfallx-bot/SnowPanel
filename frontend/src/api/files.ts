import axios from "axios";
import { ApiError, http, unwrap } from "@/lib/http";
import {
  CreateDirectoryPayload,
  CreateDirectoryResult,
  DeleteFilePayload,
  DeleteFileResult,
  ListFilesResult,
  RenameFilePayload,
  RenameFileResult,
  ReadTextFilePayload,
  ReadTextFileResult,
  WriteTextFilePayload,
  WriteTextFileResult
} from "@/types/file";
import { ApiEnvelope } from "@/types/api";

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

export function renameFile(payload: RenameFilePayload) {
  return unwrap<RenameFileResult>(http.post("/api/v1/files/rename", payload));
}

export async function downloadFile(path: string) {
  try {
    const response = await http.get("/api/v1/files/download", {
      params: { path },
      responseType: "blob"
    });
    return response.data as Blob;
  } catch (error) {
    if (!axios.isAxiosError(error)) {
      throw error;
    }

    const status = error.response?.status;
    const data = error.response?.data;
    if (data instanceof Blob) {
      try {
        const text = await data.text();
        const payload = JSON.parse(text) as Partial<ApiEnvelope<unknown>>;
        if (typeof payload.message === "string") {
          throw new ApiError(payload.message, {
            code: typeof payload.code === "number" ? payload.code : undefined,
            status,
            cause: error
          });
        }
      } catch {
        // fall through to generic error handling below
      }
    }

    throw new ApiError(error.message || "Download failed", {
      status,
      cause: error
    });
  }
}
