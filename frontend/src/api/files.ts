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
  UploadFileOptions,
  UploadFileResult,
  WriteTextFilePayload,
  WriteTextFileResult
} from "@/types/file";
import { ApiEnvelope } from "@/types/api";

export interface DownloadFilePayload {
  blob: Blob;
  fileName: string | null;
}

const uploadChunkSize = 1024 * 1024;
const uploadRetryLimit = 3;
const downloadChunkSize = 1024 * 1024;
const downloadRetryLimit = 3;

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

export function uploadFile(file: Blob, path: string, options: UploadFileOptions = {}) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("path", path);
  if (typeof options.offset === "number" && Number.isFinite(options.offset) && options.offset > 0) {
    formData.append("offset", String(options.offset));
  }
  return unwrap<UploadFileResult>(http.post("/api/v1/files/upload", formData));
}

export async function uploadFileWithRetry(file: File, path: string) {
  let offset = 0;
  let lastResult: UploadFileResult | null = null;

  while (offset < file.size || (file.size === 0 && offset === 0)) {
    const chunk = file.slice(offset, offset + uploadChunkSize);
    let success = false;
    let lastError: unknown = null;

    for (let attempt = 0; attempt < uploadRetryLimit; attempt += 1) {
      try {
        const result = await uploadFile(chunk, path, { offset });
        const advancedBytes = result.uploaded_bytes - offset;
        if (advancedBytes < 0) {
          throw new ApiError("Upload offset moved backwards", { cause: result });
        }
        if (chunk.size > 0 && advancedBytes === 0) {
          throw new ApiError("Upload did not advance", { cause: result });
        }
        offset = result.uploaded_bytes;
        lastResult = result;
        success = true;
        break;
      } catch (error) {
        lastError = error;
      }
    }

    if (!success) {
      throw lastError;
    }

    if (file.size === 0) {
      break;
    }
  }

  if (!lastResult) {
    return {
      path,
      uploaded_bytes: 0,
      total_size: 0
    } satisfies UploadFileResult;
  }

  return lastResult;
}

function parseDownloadFileName(contentDisposition: string | null | undefined) {
  if (!contentDisposition) {
    return null;
  }

  const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) {
    try {
      return decodeURIComponent(utf8Match[1].trim());
    } catch {
      return utf8Match[1].trim();
    }
  }

  const quotedMatch = contentDisposition.match(/filename="([^"]+)"/i);
  if (quotedMatch?.[1]) {
    return quotedMatch[1].trim();
  }

  const plainMatch = contentDisposition.match(/filename=([^;]+)/i);
  if (plainMatch?.[1]) {
    return plainMatch[1].trim();
  }

  return null;
}

function parseContentRangeTotal(contentRange: string | null | undefined) {
  if (!contentRange) {
    return null;
  }

  const match = contentRange.match(/^bytes\s+\d+-\d+\/(\d+)$/i);
  if (!match?.[1]) {
    return null;
  }

  const total = Number(match[1]);
  return Number.isFinite(total) && total >= 0 ? total : null;
}

async function downloadFileChunk(path: string, offset: number, limit: number) {
  const response = await http.get("/api/v1/files/download", {
    params: { path, offset, limit },
    responseType: "blob",
    headers: {
      Range: `bytes=${offset}-`
    }
  });

  return {
    blob: response.data as Blob,
    fileName: parseDownloadFileName(response.headers["content-disposition"]),
    totalSize: parseContentRangeTotal(response.headers["content-range"])
  };
}

export async function downloadFile(path: string) {
  let offset = 0;
  let totalSize: number | null = null;
  let fileName: string | null = null;
  const chunks: BlobPart[] = [];

  while (totalSize === null || offset < totalSize) {
    let success = false;
    let lastError: unknown = null;

    for (let attempt = 0; attempt < downloadRetryLimit; attempt += 1) {
      try {
        const chunk = await downloadFileChunk(path, offset, downloadChunkSize);
        if (fileName === null) {
          fileName = chunk.fileName;
        }
        if (chunk.totalSize !== null) {
          totalSize = chunk.totalSize;
        }
        if (chunk.blob.size === 0) {
          if (totalSize === null || offset < totalSize) {
            throw new ApiError("Download did not advance", { cause: chunk });
          }
          success = true;
          break;
        }
        chunks.push(chunk.blob);
        offset += chunk.blob.size;
        success = true;
        break;
      } catch (error) {
        lastError = error;
      }
    }

    if (!success) {
      if (axios.isAxiosError(lastError)) {
        const status = lastError.response?.status;
        const data = lastError.response?.data;
        if (data instanceof Blob) {
          try {
            const text = await data.text();
            const payload = JSON.parse(text) as Partial<ApiEnvelope<unknown>>;
            if (typeof payload.message === "string") {
              throw new ApiError(payload.message, {
                code: typeof payload.code === "number" ? payload.code : undefined,
                status,
                cause: lastError
              });
            }
          } catch {
            // fall through
          }
        }
      }
      throw lastError;
    }

    if (totalSize !== null && offset >= totalSize) {
      break;
    }
  }

  return {
    blob: new Blob(chunks),
    fileName
  } as DownloadFilePayload;
}
