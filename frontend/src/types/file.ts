export interface FileEntry {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  modified_at_unix: number;
}

export interface ListFilesResult {
  current_path: string;
  entries: FileEntry[];
}

export interface ReadTextFilePayload {
  path: string;
  max_bytes?: number;
  encoding?: string;
}

export interface ReadTextFileResult {
  path: string;
  content: string;
  size: number;
  truncated: boolean;
  encoding: string;
}

export interface WriteTextFilePayload {
  path: string;
  content: string;
  create_if_not_exists: boolean;
  truncate: boolean;
  encoding?: string;
}

export interface WriteTextFileResult {
  path: string;
  written_bytes: number;
}

export interface CreateDirectoryPayload {
  path: string;
  create_parents: boolean;
}

export interface CreateDirectoryResult {
  path: string;
}

export interface DeleteFilePayload {
  path: string;
  recursive: boolean;
}

export interface DeleteFileResult {
  path: string;
}

export interface RenameFilePayload {
  source_path: string;
  target_path: string;
}

export interface RenameFileResult {
  source_path: string;
  target_path: string;
  written_bytes: number;
}
